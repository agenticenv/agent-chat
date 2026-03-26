FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 go build -mod=vendor -o /server .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /server /server
EXPOSE 8080
CMD ["/server"]
