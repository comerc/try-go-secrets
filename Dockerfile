FROM golang:1.25-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /go-secrets-pipeline ./cmd

# Runtime
FROM alpine:3.23
RUN apk add --no-cache ffmpeg ca-certificates tzdata

WORKDIR /app
COPY --from=builder /go-secrets-pipeline .
COPY .env.example .env.example

RUN mkdir -p output/scripts output/audio output/videos output/logs state

ENTRYPOINT ["./go-secrets-pipeline"]
