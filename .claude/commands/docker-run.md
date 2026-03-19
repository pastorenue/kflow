# docker-run

Run, test, build, or vet the kflow project using Docker (Go is not assumed to be installed locally).

## Usage

```
/docker-run [command]
```

Where `[command]` is one of:
- `test` — run all unit tests
- `build` — build all packages
- `vet` — run `go vet`
- `up` — start the full stack (orchestrator + MongoDB + ClickHouse) via docker-compose
- `down` — stop the full stack

## Commands

**Run unit tests (no external deps):**
```bash
docker run --rm -v "$(pwd)":/workspace -w /workspace golang:1.22 go test ./...
```

**Build all packages:**
```bash
docker run --rm -v "$(pwd)":/workspace -w /workspace golang:1.22 go build ./...
```

**Run go vet:**
```bash
docker run --rm -v "$(pwd)":/workspace -w /workspace golang:1.22 go vet ./...
```

**Start full stack:**
```bash
docker compose up --build
```

**Stop full stack:**
```bash
docker compose down
```

## Notes

- Always use `golang:1.22` image to match the `go.mod` version.
- For integration tests requiring MongoDB or ClickHouse, use docker-compose to bring up dependencies first.
- The `cmd/orchestrator/` binary is the composition root; it is built by the multi-stage `Dockerfile`.
