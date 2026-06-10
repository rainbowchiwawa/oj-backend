FROM golang:latest AS builder
WORKDIR /app
COPY ./src /app/
RUN go mod download
RUN go build -o main ./

FROM alpine:latest
WORKDIR /app
USER root
RUN apk add --no-cache docker-cli gcompat
COPY --from=builder /app/main .
CMD ["./main"]