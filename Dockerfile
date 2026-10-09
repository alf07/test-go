FROM golang:1.27.1-alpine3.23 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux \
    go build -trimpath -o /app/test-go .

FROM alpine:3.23

WORKDIR /app

COPY --from=builder /app/test-go ./test-go

EXPOSE 8081

CMD ["./test-go"]