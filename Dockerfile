FROM golang:1.22 AS builder


WORKDIR /go/builder
COPY go.mod go.sum ./

RUN go mod download

COPY cmd/nriplugin/main.go cmd/nriplugin/main.go
COPY pkg/agent pkg/agent
COPY pkg/apis pkg/apis
COPY pkg/policy pkg/policy
COPY pkg/resmgr pkg/resmgr

RUN CGO_ENABLED=0 GOOS=linux go build -a -o operator cmd/nriplugin/main.go

FROM gcr.io/distroless/static:latest

WORKDIR /
COPY --from=builder /go/builder/operator .

ENTRYPOINT ["/operator"]
