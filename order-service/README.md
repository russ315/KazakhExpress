# KazakhExpress Order Service

Тема проекта: национальный казахский аналог AliExpress.

Этот репозиторий содержит основу для `order service` на Go. Сервис отвечает за создание заказов, просмотр списка заказов, получение заказа по ID и обновление статуса заказа.

## Что добавлено

- `go.mod` - модуль Go для order service.
- `cmd/order-service/main.go` - точка запуска сервиса.
- `internal/order/model.go` - основные модели заказа, товара и статусов.
- `internal/order/repository.go` - интерфейс репозитория и временное хранение заказов в памяти.
- `internal/order/service.go` - бизнес-логика создания заказа, подсчета суммы и обновления статуса.
- `internal/http/handler.go` - HTTP API для работы с заказами.

## Запуск

```powershell
go run ./cmd/order-service
```

По умолчанию сервис запускается на порту `8080`. Можно изменить порт через переменную окружения:

```powershell
$env:ORDER_SERVICE_PORT="8081"
go run ./cmd/order-service
```

## API

### Проверка сервиса

```http
GET /health
```

### Создание заказа

```http
POST /orders
```

Пример тела запроса:

```json
{
  "customer_id": "customer-1",
  "items": [
    {
      "product_id": "product-1",
      "name": "Kazakh handmade shapan",
      "quantity": 1,
      "price_kzt": 25000
    }
  ]
}
```

### Получение заказов

```http
GET /orders
GET /orders/{id}
```

### Обновление статуса

```http
PATCH /orders/{id}/status
```

Пример тела запроса:

```json
{
  "status": "paid"
}
```

Доступные статусы:

- `created`
- `paid`
- `shipped`
- `completed`
- `canceled`

## Что можно добавить дальше

- Подключение PostgreSQL вместо временного хранения в памяти.
- Таблицы и миграции для заказов и товаров заказа.
- Интеграция с cart service, product service, payment service и delivery service.
- Проверка наличия товара перед созданием заказа.
- История изменения статусов заказа.
- Авторизация пользователя через JWT.
- gRPC или message broker для общения между микросервисами.
- Тесты для service layer и HTTP handlers.
