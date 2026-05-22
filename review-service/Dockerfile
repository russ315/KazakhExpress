FROM golang:1.25-alpine AS builder

ENV GOPROXY=https://goproxy.cn,https://proxy.golang.org,direct

WORKDIR /src
COPY kazakhexpress-proto /kazakhexpress-proto
COPY KazakhExpress/review-service/go.mod KazakhExpress/review-service/go.sum* ./
RUN go mod download
COPY KazakhExpress/review-service ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/review-service ./cmd/review-service

FROM alpine:3.20
WORKDIR /app
COPY --from=builder /out/review-service /app/review-service
EXPOSE 9096
ENTRYPOINT ["/app/review-service"]
