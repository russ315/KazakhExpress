FROM golang:1.25-alpine AS build

ENV GOPROXY=https://goproxy.cn,https://proxy.golang.org,direct

WORKDIR /src
COPY kazakhexpress-proto ./kazakhexpress-proto
COPY KazakhExpress/order-service ./KazakhExpress/order-service
WORKDIR /src/KazakhExpress/order-service
RUN go mod download
RUN go build -o /out/order-service ./cmd/order-service

FROM alpine:3.22
WORKDIR /app
COPY --from=build /out/order-service /app/order-service
EXPOSE 8082 9092
ENTRYPOINT ["/app/order-service"]
