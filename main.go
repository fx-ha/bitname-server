package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

const (
	maxBody = 1 << 20 // 1 MiB, matches orchestrator commitMaxBody
)

var dataPath = envOr("BITNAME_DATA", "DATA.json")

func listenAddr() string {
	if p := os.Getenv("PORT"); p != "" {
		return "0.0.0.0:" + p
	}
	return "0.0.0.0:6002"
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

type rpcRequest struct {
	JSONRPC string            `json:"jsonrpc"`
	ID      json.RawMessage   `json:"id"`
	Method  string            `json:"method"`
	Params  json.RawMessage   `json:"params"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

func loadData() (json.RawMessage, error) {
	raw, err := os.ReadFile(dataPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", dataPath, err)
	}
	if len(raw) > maxBody {
		return nil, fmt.Errorf("%s is %d bytes, over %d byte cap", dataPath, len(raw), maxBody)
	}
	var v map[string]any
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, fmt.Errorf("%s must hold a JSON object: %w", dataPath, err)
	}
	canonical, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return canonical, nil
}

func writeError(w http.ResponseWriter, id json.RawMessage, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(rpcResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &rpcError{Code: code, Message: msg},
	})
}

func handleRPC(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, json.RawMessage("null"), -32600, "only POST is supported")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBody)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, json.RawMessage("null"), -32700, "cannot read request body")
		return
	}
	var req rpcRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, json.RawMessage("null"), -32700, "invalid JSON-RPC request")
		return
	}
	if req.Method != "bitname_commit" {
		writeError(w, req.ID, -32601, "method not found: "+req.Method)
		return
	}
	// params ([null] on first call, [<hex>] on recursive calls) are
	// intentionally ignored: always serve the same commitment object.
	result, err := loadData()
	if err != nil {
		log.Printf("loadData: %v", err)
		writeError(w, req.ID, -32603, "cannot load commitment data")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(rpcResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	})
}

func main() {
	if len(os.Args) > 1 {
		dataPath = os.Args[1]
	}
	if _, err := loadData(); err != nil {
		log.Fatalf("startup: %v", err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", handleRPC)
	addr := listenAddr()
	srv := &http.Server{
		Addr: addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
	}
	log.Printf("serving bitname_commit from %s on http://%s/", dataPath, addr)
	log.Fatal(srv.ListenAndServe())
}
