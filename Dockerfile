FROM golang:1.15-alpine AS builder
WORKDIR /app
RUN apk add gcc g++ --no-cache
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -ldflags '-w -extldflags "-static"' -o app ./grpc/server
RUN chmod +x app
RUN chmod +r ./config/config.yml

FROM scratch
WORKDIR /app
COPY --from=builder /app/config/config.yml .
COPY --from=builder /app/app  .

ENTRYPOINT ["/app/app"]