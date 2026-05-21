FROM golang:1.25-alpine AS build

WORKDIR /src/order-service
COPY order-service ./
RUN go mod download
RUN go build -o /out/order-service ./cmd/order-service

FROM alpine:3.22
WORKDIR /app
COPY --from=build /out/order-service /app/order-service
EXPOSE 8082 9092
ENTRYPOINT ["/app/order-service"]
