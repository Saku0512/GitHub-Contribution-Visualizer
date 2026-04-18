FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/api ./cmd/api

FROM alpine:3.22

WORKDIR /app

COPY --from=builder /out/api /usr/local/bin/api

EXPOSE 8080

ENV PORT=8080

CMD ["api"]
