FROM golang:1.22 AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN go build -o /orchestrator ./cmd/orchestrator/

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /orchestrator /orchestrator
USER nonroot:nonroot
ENTRYPOINT ["/orchestrator"]
