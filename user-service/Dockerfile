FROM golang:1.25-alpine AS build

ENV GOPROXY=https://goproxy.cn,https://proxy.golang.org,direct

WORKDIR /src
COPY kazakhexpress-proto ./kazakhexpress-proto
COPY KazakhExpress/user-service ./KazakhExpress/user-service

WORKDIR /src/KazakhExpress/user-service
RUN go mod download
RUN go build -o /out/user-service ./cmd/user-service

FROM alpine:3.22
WORKDIR /app
COPY --from=build /out/user-service /app/user-service
EXPOSE 8081 50051
ENTRYPOINT ["/app/user-service"]
