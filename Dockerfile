FROM golang:1-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o mini-proxy .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/mini-proxy .
EXPOSE 8080
CMD ["./mini-proxy"]