# Архитектура KazakhExpress

## Общая схема системы

```mermaid
graph TB
    Client["Браузер / Frontend :5173"]
    Nginx["Nginx Reverse Proxy<br/>kazexp.maqsatto.dev"]
    
    subgraph "Public Layer"
        APIGW["API Gateway :8080<br/>Gin HTTP + Rate Limiter"]
    end

    subgraph "Service Layer (gRPC)"
        US["User Service<br/>:50051 gRPC / :8081 HTTP"]
        PS["Product Service<br/>:9095 gRPC / :8084 HTTP"]
        OS["Order Service<br/>:9092 gRPC / :8082 HTTP"]
        PMS["Payment Service<br/>:9093 gRPC / :8083 HTTP"]
        RS["Review Service<br/>:9096 gRPC / :9106 Metrics"]
        SMTP["SMTP Service<br/>:9094 gRPC / :9104 Metrics"]
    end

    subgraph "Data Layer"
        PG[("PostgreSQL 17<br/>kazakhexpress")]
        REDIS[("Redis 7<br/>Кеш + Rate Limit + Blacklist")]
        MINIO[("MinIO<br/>S3 Object Storage")]
    end

    subgraph "Message Broker"
        NATS["NATS JetStream<br/>:4222"]
    end

    subgraph "Observability"
        PROM["Prometheus :9090"]
        GRAF["Grafana :3000"]
        LOKI["Loki :3100"]
        TEMPO["Tempo :3200"]
        OTEL["OTel Collector"]
    end

    Client --> Nginx
    Nginx --> |"/api/*"| APIGW
    Nginx --> |"/"| Client
    Nginx --> |"/metrics/*"| GRAF

    APIGW -->|gRPC| US
    APIGW -->|gRPC| PS
    APIGW -->|gRPC| OS
    APIGW -->|gRPC| PMS
    APIGW -->|gRPC| RS
    APIGW --> REDIS

    US -->|gRPC| SMTP
    PMS -->|gRPC| SMTP
    
    US --> PG
    PS --> PG
    OS --> PG
    PMS --> PG
    RS --> PG

    US --> REDIS
    OS --> REDIS
    PMS --> REDIS
    RS --> REDIS

    PS --> MINIO

    US -->|publish| NATS
    PS -->|publish| NATS
    OS -->|publish + consume| NATS
    PMS -->|publish| NATS
    RS -->|publish| NATS

    PS --> PROM
    US --> PROM
    OS --> PROM
    PMS --> PROM
    RS --> PROM
    APIGW --> PROM
    NATS -->|NATS Exporter| PROM
    PROM --> GRAF
    LOKI --> GRAF
    TEMPO --> OTEL --> GRAF
```

## Коммуникация между сервисами

```mermaid
graph LR
    subgraph "Внешний мир"
        HTTP["HTTP REST"]
    end
    subgraph "Шлюз"
        GW["API Gateway<br/>:8080"]
    end
    subgraph "gRPC вызовы"
        GW -->|"auth/register/login"| US1["User Service"]
        GW -->|"products CRUD"| PS1["Product Service"]
        GW -->|"orders CRUD"| OS1["Order Service"]
        GW -->|"payments"| PMS1["Payment Service"]
        GW -->|"reviews"| RS1["Review Service"]
        US1 -->|"send email"| SMTP1["SMTP Service"]
        PMS1 -->|"send receipt"| SMTP1
    end
    HTTP --> GW
```

## Поток запроса (на примере оформления заказа)

```mermaid
sequenceDiagram
    actor User as Пользователь
    participant FE as Frontend
    participant GW as API Gateway
    participant US as User Service
    participant OS as Order Service
    participant PMS as Payment Service
    participant RS as Review Service
    participant SMTP as SMTP Service
    participant NATS as NATS JetStream
    participant PG as PostgreSQL
    participant REDIS as Redis

    User->>FE: Регистрация
    FE->>GW: POST /auth/register
    GW->>US: gRPC Register()
    US->>PG: INSERT users
    US->>REDIS: blacklist + rate limit
    US-->>NATS: publish user.created
    US->>SMTP: gRPC SendEmail (welcome)
    US-->>GW: JWT + Refresh Token
    GW-->>FE: 200 + tokens
    FE-->>User: Успех

    User->>FE: Просмотр товаров
    FE->>GW: GET /products
    GW->>PS: gRPC ListProducts()
    PS->>PG: SELECT products
    PS-->>GW: список товаров
    GW-->>FE: 200 + products

    User->>FE: Оформление заказа
    FE->>GW: POST /orders
    GW->>OS: gRPC CreateOrder()
    OS->>PG: INSERT orders + items
    OS-->>NATS: publish order.created
    OS-->>GW: order created
    GW-->>FE: 200 + orderId

    FE->>GW: POST /payment
    GW->>PMS: gRPC CreatePayment()
    PMS->>REDIS: check idempotency key
    PMS->>PG: INSERT payments
    PMS->>PMS: call mock provider
    alt Payment Success
        PMS-->>NATS: publish payment.succeeded
        NATS-->>OS: consume payment.succeeded
        OS->>PG: UPDATE order status = paid
        OS-->>NATS: publish order.completed
        PMS->>SMTP: gRPC SendPaymentReceipt
        PMS-->>GW: payment succeeded
        GW-->>FE: 200 + success
    else Payment Failed
        PMS-->>NATS: publish payment.failed
        NATS-->>OS: consume payment.failed
        OS->>PG: UPDATE order status = payment_failed
        PMS->>SMTP: gRPC SendPaymentFailure
        PMS-->>GW: payment failed
        GW-->>FE: 200 + failed
    end

    User->>FE: Оставить отзыв
    FE->>GW: POST /products/:id/reviews
    GW->>RS: gRPC CreateReview()
    RS->>PG: INSERT reviews
    RS->>REDIS: invalidate rating cache
    RS-->>NATS: publish review.created
    RS-->>GW: review created
    GW-->>FE: 201
```

## Аутентификация и JWT

```mermaid
sequenceDiagram
    actor User as Пользователь
    participant GW as API Gateway
    participant US as User Service
    participant PG as PostgreSQL
    participant REDIS as Redis

    User->>GW: POST /auth/login (email + password)
    GW->>US: gRPC Login()
    US->>REDIS: check rate limit (5 попыток / 15 мин)
    US->>PG: SELECT user by email
    US->>US: bcrypt verify password
    alt успех
        US->>US: generate JWT access (7d) + refresh (30d)
        US->>PG: INSERT refresh_token (SHA256 hash)
        US-->>GW: JWT + RefreshToken
        GW-->>User: 200 + tokens
    else превышен лимит
        US-->>GW: 429 Too Many Requests
        GW-->>User: 429
    else неверный пароль
        US->>REDIS: increment login attempts
        US-->>GW: 401 Unauthorized
        GW-->>User: 401
    end

    User->>GW: POST /auth/refresh (refresh_token)
    GW->>US: gRPC RefreshToken()
    US->>PG: DELETE old refresh_token
    US->>US: generate new JWT + refresh token
    US->>PG: INSERT new refresh_token
    US-->>GW: new tokens

    User->>GW: POST /auth/logout (JWT)
    GW->>US: gRPC Logout()
    US->>REDIS: SET jti -> blacklisted (TTL 7d)
    US->>PG: DELETE refresh_token
    US-->>GW: 200 OK
```

## Топология NATS JetStream

```mermaid
graph TB
    subgraph "Event Bus NATS"
        NATS["NATS JetStream"]
    end
    
    US["User Service"] -->|"user.created"| NATS
    US -->|"user.updated"| NATS
    
    PS["Product Service"] -->|"product.created"| NATS
    PS -->|"stock.updated"| NATS
    PS -->|"stock.reserved"| NATS
    PS -->|"stock.released"| NATS
    
    OS["Order Service"] -->|"order.created"| NATS
    OS -->|"order.completed"| NATS
    OS -->|"order.cancelled"| NATS
    NATS -->|"payment.succeeded"| OS
    NATS -->|"payment.failed"| OS
    
    PMS["Payment Service"] -->|"payment.created"| NATS
    PMS -->|"payment.succeeded"| NATS
    PMS -->|"payment.failed"| NATS
    PMS -->|"payment.refunded"| NATS
    PMS -->|"payment.cancelled"| NATS
    
    RS["Review Service"] -->|"review.created"| NATS
    RS -->|"review.updated"| NATS
    RS -->|"review.deleted"| NATS
    RS -->|"rating.updated"| NATS
```

## Схема базы данных PostgreSQL

```mermaid
erDiagram
    users {
        uuid id PK
        varchar email UK
        varchar password_hash
        varchar full_name
        varchar phone
        varchar role
        timestamp created_at
        timestamp updated_at
    }
    refresh_tokens {
        uuid id PK
        uuid user_id FK
        varchar token_hash
        timestamp expires_at
        timestamp created_at
    }
    token_blacklist {
        varchar jti PK
        timestamp expires_at
    }
    
    products {
        uuid id PK
        varchar name
        text description
        decimal price
        int stock
        varchar category
        varchar status
        timestamp created_at
    }
    product_images {
        uuid id PK
        uuid product_id FK
        varchar object_key
        bool is_primary
        timestamp created_at
    }
    
    orders {
        uuid id PK
        uuid user_id FK
        varchar status
        decimal total_amount
        timestamp created_at
        timestamp updated_at
    }
    order_items {
        uuid id PK
        uuid order_id FK
        uuid product_id FK
        int quantity
        decimal unit_price
    }
    order_status_history {
        uuid id PK
        uuid order_id FK
        varchar from_status
        varchar to_status
        timestamp created_at
    }
    
    payments {
        uuid id PK
        uuid order_id FK
        uuid user_id FK
        varchar idempotency_key
        decimal amount
        varchar currency
        varchar status
        varchar provider_tx_id
        timestamp created_at
        timestamp updated_at
    }
    payment_events {
        uuid id PK
        uuid payment_id FK
        varchar event_type
        jsonb payload
        timestamp created_at
    }
    refunds {
        uuid id PK
        uuid payment_id FK
        decimal amount
        varchar reason
        varchar status
        timestamp created_at
    }
    
    reviews {
        uuid id PK
        uuid product_id FK
        uuid user_id FK
        int rating
        text comment
        timestamp created_at
        timestamp updated_at
    }

    users ||--o{ refresh_tokens : has
    users ||--o{ orders : places
    users ||--o{ reviews : writes
    products ||--o{ product_images : has
    products ||--o{ order_items : includes
    products ||--o{ reviews : has
    orders ||--o{ order_items : contains
    orders ||--o{ order_status_history : tracks
    payments ||--o{ payment_events : logs
    payments ||--o{ refunds : has
```

## Использование Redis по сервисам

```mermaid
graph TB
    subgraph "Redis 7"
        RATE["Rate Limiting<br/>SLIDING_WINDOW:{ip}:{route}"]
        BL["Token Blacklist<br/>blacklist:{jti}"]
        LOGIN["Login Attempts<br/>login_attempts:{email}"]
        IDEM["Idempotency<br/>idempotency:{key}"]
        STAT["Status Cache<br/>order_status:{id}"]
        RAT["Rating Cache<br/>rating:{product_id}"]
    end
    
    GW["API Gateway"] --> RATE
    US["User Service"] --> BL
    US --> LOGIN
    PMS["Payment Service"] --> IDEM
    OS["Order Service"] --> STAT
    RS["Review Service"] --> RAT
```

## Инфраструктура и деплой

```mermaid
graph TB
    subgraph "Production Server"
        subgraph "Docker Compose (18 контейнеров)"
            direction TB
            APP["api-gateway :8080<br/>user-service<br/>product-service<br/>order-service<br/>payment-service<br/>review-service<br/>smtp-service<br/>frontend"]
            DB["postgres :5432<br/>redis :6379<br/>minio :9000<br/>nats :4222"]
            OBS["prometheus :9090<br/>grafana :3000<br/>loki :3100<br/>tempo :3200<br/>otel-collector"]
        end
        NGINX["Nginx<br/>HTTPS (Let's Encrypt)"]
        CERT["Certbot<br/>SSL Auto-Renewal"]
    end
    
    INTERNET["Internet"] -->|"HTTPS :443"| NGINX
    NGINX -->|"/api/*"| APP
    NGINX -->|"/"| APP
    NGINX -->|"/metrics/*"| OBS
    CERT --> NGINX
```

## Технологический стек

| Компонент | Технология | Назначение |
|-----------|-----------|------------|
| **Язык** | Go 1.25 | Все микросервисы |
| **HTTP Framework** | Gin | API Gateway и HTTP endpoints |
| **API Gateway** | Custom Go/Gin | Единая публичная точка входа |
| **Service Mesh** | gRPC | Внутренняя коммуникация сервисов |
| **Event Bus** | NATS 2.11 + JetStream | Асинхронные события |
| **База данных** | PostgreSQL 17 | Персистентное хранилище |
| **Кеш** | Redis 7 | Rate limiting, blacklist, idempotency, кеш |
| **Объектное хранилище** | MinIO (S3) | Изображения товаров |
| **Email** | Resend API + SMTP fallback | Отправка писем |
| **Аутентификация** | JWT (HS256) + bcrypt + Refresh Tokens | Auth |
| **Observability** | Prometheus + Grafana + Loki + Tempo + OTel | Мониторинг, логи, трейсинг |
| **Фронтенд** | React 19 + TypeScript + Vite | UI marketplace |
| **CI/CD** | GitHub Actions (4 workflows) | Линтинг, тесты, proto generation |
| **Деплой** | Docker Compose + Nginx + Let's Encrypt | Продакшн |
| **API Spec** | OpenAPI 3.0.3 | Документация API |

## Структура репозитория

```
KazakhExpress/
├── api-gateway/          # Единый HTTP шлюз (Gin)
│   └── internal/
│       ├── gateway/          # Router, rate limiter (Redis)
│       ├── orderservice/     # gRPC client для Order Service
│       ├── paymentservice/   # gRPC client для Payment Service
│       ├── productservice/   # gRPC client для Product Service
│       ├── reviewservice/    # gRPC client для Review Service
│       └── userservice/      # gRPC client для User Service
├── user-service/         # Аутентификация, профили, JWT
│   └── internal/
│       ├── email/            # gRPC client -> smtp-service
│       ├── grpc/             # gRPC server
│       ├── http/             # HTTP handler
│       ├── messaging/        # NATS publisher
│       ├── redis/            # Redis client
│       └── user/             # Бизнес-логика
├── product-service/      # Товары, сток, изображения
│   └── internal/
│       ├── grpcapi/          # gRPC server
│       ├── http/             # HTTP handler
│       ├── messaging/        # NATS publisher
│       ├── product/          # Бизнес-логика
│       └── storage/          # MinIO client
├── order-service/        # Заказы, статусы
│   └── internal/
│       ├── cache/            # Redis status cache
│       ├── grpcapi/          # gRPC server
│       ├── http/             # HTTP handler
│       ├── messaging/        # NATS publisher + consumer
│       └── order/            # Бизнес-логика
├── payment-service/      # Платежи, idempotency, refunds
│   └── internal/
│       ├── cache/            # Redis idempotency store
│       ├── email/            # gRPC client -> smtp-service
│       ├── grpcapi/          # gRPC server
│       ├── http/             # HTTP handler
│       ├── messaging/        # NATS publisher
│       ├── payment/          # Бизнес-логика
│       └── provider/         # Mock payment provider
├── review-service/       # Отзывы, рейтинг
│   └── internal/
│       ├── cache/            # Redis rating cache
│       ├── grpcapi/          # gRPC server
│       ├── messaging/        # NATS publisher
│       └── review/           # Бизнес-логика
├── smtp-service/         # Отправка email (Resend + SMTP fallback)
│   └── internal/
│       ├── grpcapi/          # gRPC server
│       ├── smtp/             # SMTP sender
│       └── smtpapp/          # Wiring
├── frontend/             # React + TypeScript (Vite)
├── infra/                # Observability конфиги
│   ├── grafana/              # Dashboard JSON + provisioning
│   ├── prometheus/           # prometheus.yml
│   ├── loki/                 # loki.yml
│   ├── tempo/                # tempo.yml
│   └── otel-collector/       # config.yml
├── deploy/               # Продакшн деплой
│   ├── nginx/                # kazexp.maqsatto.dev.conf
│   └── setup-ubuntu.sh       # автоматический деплой
├── scripts/              # Тесты и demo скрипты
└── docs/                 # Документация
```

## Ключевые особенности архитектуры

1. **Единая точка входа** — API Gateway обрабатывает все внешние запросы, внутренние сервисы не暴露 наружу
2. **gRPC communication** — быстрая бинарная сериализация между сервисами
3. **Асинхронные события через NATS** — слабая связанность сервисов
4. **Graceful degradation** — при недоступности NATS/Redis сервисы продолжают работу
5. **Idempotency платежей** — Redis предотвращает дублирование платежей
6. **Rate limiting в 2 слоях** — на уровне Gateway и User Service (login)
7. **Полный observability stack** — Prometheus метрики, Loki логи, Tempo трейсинг, Grafana дашборды
