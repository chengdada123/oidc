FROM golang:1.22 AS builder
WORKDIR /src

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/oidc-bridge ./cmd/bridge

FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=builder /out/oidc-bridge /app/oidc-bridge
COPY --from=builder /src/internal/web/templates /app/internal/web/templates
ENV PORT=8080
EXPOSE 8080
CMD ["/app/oidc-bridge"]
