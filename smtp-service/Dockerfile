FROM golang:1.25-alpine AS build

WORKDIR /src
COPY kazakhexpress-proto ./kazakhexpress-proto
COPY KazakhExpress/smtp-service ./KazakhExpress/smtp-service

WORKDIR /src/KazakhExpress/smtp-service
RUN go mod download
RUN go build -o /out/smtp-service ./cmd/smtp-service

FROM alpine:3.22
WORKDIR /app
COPY --from=build /out/smtp-service /app/smtp-service
EXPOSE 9094
ENTRYPOINT ["/app/smtp-service"]

