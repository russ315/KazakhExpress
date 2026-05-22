FROM golang:1.25-alpine AS build

WORKDIR /src
COPY KazakhExpress/review-service ./KazakhExpress/review-service

WORKDIR /src/KazakhExpress/review-service
RUN go mod download
RUN go build -o /out/review-service ./cmd/review-service

FROM alpine:3.22
WORKDIR /app
COPY --from=build /out/review-service /app/review-service
EXPOSE 9095
ENTRYPOINT ["/app/review-service"]
