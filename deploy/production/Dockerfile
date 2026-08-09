FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETOS=linux
ARG TARGETARCH=amd64
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" -o /out/identity-rpc ./service/identity/rpc && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" -o /out/app-api ./service/app/api && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" -o /out/db-migrate ./tools/db-migrate

FROM alpine:3.22 AS db-migrate

RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /out/db-migrate /usr/local/bin/db-migrate

USER 65532:65532
ENTRYPOINT ["/usr/local/bin/db-migrate"]

FROM alpine:3.22 AS identity-rpc

RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /out/identity-rpc /usr/local/bin/identity-rpc
COPY deploy/production/config/identity-rpc.yaml /etc/hospital/identity-rpc.yaml

USER 65532:65532
ENTRYPOINT ["/usr/local/bin/identity-rpc", "-f", "/etc/hospital/identity-rpc.yaml"]

FROM alpine:3.22 AS app-api

RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /out/app-api /usr/local/bin/app-api
COPY deploy/production/config/app-api.yaml /etc/hospital/app-api.yaml

USER 65532:65532
ENTRYPOINT ["/usr/local/bin/app-api", "-f", "/etc/hospital/app-api.yaml"]
