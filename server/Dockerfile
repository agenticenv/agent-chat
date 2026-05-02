FROM golang:1.24-alpine AS builder
RUN apk --no-cache add ca-certificates git

WORKDIR /app

COPY server/go.mod server/go.sum ./
RUN go mod download

COPY server/ .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /app-bin .

FROM alpine:3.21
RUN apk --no-cache add ca-certificates
COPY --from=builder /app-bin /app-bin

# Build-time default mode. Override at runtime via APP_MODE env or --mode flag.
ARG MODE=server
ENV APP_MODE=$MODE

EXPOSE 8080
USER nobody
ENTRYPOINT ["/app-bin"]
