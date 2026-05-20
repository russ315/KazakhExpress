FROM golang:1.25-alpine AS build

WORKDIR /src
COPY kazakhexpress-proto ./kazakhexpress-proto
COPY KazakhExpress/api-gateway ./KazakhExpress/api-gateway

WORKDIR /src/KazakhExpress/api-gateway
RUN go mod download
RUN go build -o /out/api-gateway ./cmd/api-gateway

FROM alpine:3.22
WORKDIR /app
COPY --from=build /out/api-gateway /app/api-gateway
EXPOSE 8080
ENTRYPOINT ["/app/api-gateway"]

