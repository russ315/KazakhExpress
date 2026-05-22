module kazakhexpress/user-service

go 1.25.0

require (
	github.com/golang-jwt/jwt/v5 v5.2.0
	github.com/google/uuid v1.6.0
	github.com/lib/pq v1.10.9
	github.com/maqsatto/kazakhexpress-proto v0.0.0-20260522043009-8ad032f507fa
	github.com/rabbitmq/amqp091-go v1.9.0
	github.com/redis/go-redis/v9 v9.3.0
	golang.org/x/crypto v0.48.0
	google.golang.org/grpc v1.81.1
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	golang.org/x/net v0.51.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260226221140-a57be14db171 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace github.com/maqsatto/kazakhexpress-proto => ../../kazakhexpress-proto
