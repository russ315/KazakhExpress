FROM golang:1.25-alpine AS builder

ENV GOPROXY=https://goproxy.cn,https://proxy.golang.org,direct

WORKDIR /src
COPY kazakhexpress-proto /kazakhexpress-proto
COPY KazakhExpress/product-service/go.mod KazakhExpress/product-service/go.sum* ./
RUN go mod download
COPY KazakhExpress/product-service ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/product-service ./cmd/product-service

FROM alpine:3.20
WORKDIR /app
COPY --from=builder /out/product-service /app/product-service
EXPOSE 8084 9095
ENTRYPOINT ["/app/product-service"]
