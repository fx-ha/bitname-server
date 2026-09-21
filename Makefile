BINARY := bitname-server
PORT ?= 6002

.PHONY: build build-linux-amd64 run smoke vet clean

build:
	go build -o $(BINARY) .

build-linux-amd64:
	GOOS=linux GOARCH=amd64 go build -o $(BINARY) .

run: build
	PORT=$(PORT) ./$(BINARY) DATA.json

smoke:
	@curl -s -X POST http://127.0.0.1:$(PORT)/ \
		-H 'Content-Type: application/json' \
		-d '{"jsonrpc":"2.0","id":"smoke","method":"bitname_commit","params":[null]}' \
	| grep -q '"result":{[^}]' \
	&& echo "OK: bitname_commit answers with a result object" \
	|| (echo "FAIL: unexpected answer" >&2; exit 1)

vet:
	go vet ./...

clean:
	rm -f $(BINARY)
