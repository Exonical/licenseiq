FROM golang:1.26-bookworm AS builder

WORKDIR /src

ARG VERSION=dev
ARG COMMIT=none
ARG DATE=unknown

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w \
    -X github.com/Exonical/licenseiq/backend/internal/version.Version=${VERSION} \
    -X github.com/Exonical/licenseiq/backend/internal/version.Commit=${COMMIT} \
    -X github.com/Exonical/licenseiq/backend/internal/version.Date=${DATE}" \
    -o /out/licenseiq ./cmd/licenseiq

FROM alpine:3.21

RUN addgroup -S licenseiq && adduser -S -G licenseiq -u 65532 licenseiq \
    && apk add --no-cache ca-certificates wget

WORKDIR /app
COPY --from=builder /out/licenseiq /usr/local/bin/licenseiq

USER 65532:65532

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget -qO- http://127.0.0.1:8080/health >/dev/null || exit 1

ENTRYPOINT ["/usr/local/bin/licenseiq"]
