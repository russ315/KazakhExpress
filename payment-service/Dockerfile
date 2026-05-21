FROM golang:1.25-alpine AS build

WORKDIR /src
COPY kazakhexpress-proto ./kazakhexpress-proto
COPY KazakhExpress/payment-service ./KazakhExpress/payment-service

WORKDIR /src/KazakhExpress/payment-service
RUN go mod download
RUN go build -o /out/payment-service ./cmd/payment-service

FROM alpine:3.22
WORKDIR /app
COPY --from=build /out/payment-service /app/payment-service
EXPOSE 8083 9093
ENTRYPOINT ["/app/payment-service"]

