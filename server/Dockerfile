FROM golang:latest AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY ./src /app/src/
RUN go build -o main ./src

FROM alpine:latest
WORKDIR /app
RUN apk add --no-cache docker-cli gcompat
COPY --from=builder /app/main .
CMD ["./main"]