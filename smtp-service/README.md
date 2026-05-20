# KazakhExpress SMTP Service

Shared SMTP microservice for KazakhExpress services. Other services call it through gRPC instead of each service owning its own SMTP implementation.

## gRPC

```txt
SMTP_GRPC_PORT=9094
kazakhexpress.smtp.v1.SMTPService
```

Methods:

```txt
SendEmail
SendPaymentReceipt
SendPaymentRefund
SendPaymentFailure
```

## Environment

```powershell
$env:SMTP_GRPC_PORT="9094"
$env:SMTP_HOST="smtp.gmail.com"
$env:SMTP_PORT="587"
$env:SMTP_USERNAME="your-email@gmail.com"
$env:SMTP_PASSWORD="your-app-password"
$env:SMTP_FROM="noreply@kazakhexpress.kz"
```

If username/password are empty, the service accepts requests and skips the network send. This makes local demos and tests easy.

## Run

```powershell
go run ./cmd/smtp-service
```

## Test

```powershell
go test ./...
go vet ./...
```
