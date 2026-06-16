FROM golang:1.23-bookworm AS builder
WORKDIR /src

COPY . .
RUN go build -o /out/oidc-bridge ./cmd/bridge

FROM debian:bookworm-slim
WORKDIR /app
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*
COPY --from=builder /out/oidc-bridge /app/oidc-bridge
COPY --from=builder /src/internal/web/templates /app/internal/web/templates
ENV PORT=8080
EXPOSE 8080
CMD ["/app/oidc-bridge"]
