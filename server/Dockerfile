FROM golang:1.24-alpine AS builder
RUN apk --no-cache add git
WORKDIR /app

# Copy the SDK from the sibling directory (passed as additional context)
COPY --from=sdk . /sdk

# Copy server source
COPY server/ .

# Point go.mod at the local SDK copy so we don't need git credentials
RUN go mod edit -replace github.com/vvsynapse/agent-sdk-go=/sdk
RUN go mod tidy
RUN CGO_ENABLED=0 go build -o /server .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /server /server
EXPOSE 8080
CMD ["/server"]
