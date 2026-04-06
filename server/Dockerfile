FROM golang:1.24-alpine AS builder
RUN apk --no-cache add ca-certificates git

WORKDIR /app

COPY server/go.mod server/go.sum ./
RUN go mod download

COPY server/ .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /server .

FROM alpine:3.21
RUN apk --no-cache add ca-certificates
COPY --from=builder /server /server
EXPOSE 8080
USER nobody
CMD ["/server"]
