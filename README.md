# bitname-server

Minimal data server for a [Bitnames](https://github.com/LayerTwo-Labs/plain-bitnames)
bitname. It serves your public record over JSON-RPC so the Bitnames
orchestrator can hash it into your on-chain commitment.

## How it works

The chain never stores your domain or your data. It stores only:

* `commitment` — `BLAKE3` hash of your record in JCS canonical form (RFC 8785)
* `socketAddrV4` / `socketAddrV6` — the `IP:port` that served the record

When you enter your domain in Bitwindow (`Quick Lookup With Domain`) and click
`Get IP addresses`, the orchestrator resolves your domain to its public IPs,
then calls this server:

```
POST http://<your-IP>:6002/
Content-Type: application/json
{"jsonrpc":"2.0","id":"orchestrator","method":"bitname_commit","params":[null]}
```

and expects:

```
{"jsonrpc":"2.0","id":"orchestrator","result":{...your record...}}
```

The returned object is hashed; that digest becomes the commitment you register.
Any later change to the record changes the hash, so finalize your data
**before** registering (after that, changes require a `BitNameUpdate` tx).

## Quick start

```bash
cp DATA.example.json DATA.json
# edit DATA.json — it must stay a JSON object, < 1 MiB
```

### Option A: native binary (needs Go)

```bash
make run              # listens on 0.0.0.0:6002
```

### Option B: Docker (no Go needed)

```bash
docker compose up -d
docker compose logs -f
```

### Verify

```bash
make smoke            # POSTs bitname_commit, checks the envelope
```

or manually:

```bash
curl -X POST http://127.0.0.1:6002/ \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":"orchestrator","method":"bitname_commit","params":[null]}'
```

## Configuration

| Source | Purpose | Default |
|---|---|---|
| `DATA.json` (or `BITNAME_DATA` env / first CLI arg) | record served as `result` | `DATA.json` |
| `PORT` env | listen port | `6002` |

`DATA.json` is re-read on every request, so edits apply without a restart.

## Deploy on a VPS (Debian, systemd)

Prerequisites: domain `A`/`AAAA` record pointing at the VM, firewall open for
`tcp:6002` (GCP VPC rule or equivalent), plain HTTP — no TLS redirect on
this port, the orchestrator refuses redirects.

```bash
# cross-compile on your machine (amd64 example):
make build-linux-amd64

# copy (first create the dir with an owner you can write to):
ssh <user>@<host> 'sudo mkdir -p /opt/bitname && sudo chown -R <user>:<user> /opt/bitname'
scp bitname-server DATA.json <user>@<host>:/opt/bitname/

# on the VPS, install the service (adjust User=):
sudo tee /etc/systemd/system/bitname.service > /dev/null <<'EOF'
[Unit]
Description=Bitname commitment server
After=network-online.target
Wants=network-online.target

[Service]
ExecStart=/opt/bitname/bitname-server /opt/bitname/DATA.json
Restart=always
RestartSec=3
User=<user>
NoNewPrivileges=true

[Install]
WantedBy=multi-user.target
EOF
sudo systemctl daemon-reload
sudo systemctl enable --now bitname
```

Then in Bitwindow enter your domain and click `Get IP addresses`.
