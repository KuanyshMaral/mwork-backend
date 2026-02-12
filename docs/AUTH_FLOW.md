# Auth Flow (актуальный)

Ниже описан фактический flow, соответствующий смонтированным роутам `auth`.

## 1) Register
`POST /api/v1/auth/register`

- Возвращает `201`
- **Токены не выдаются**
- В `data` есть `verification_sent=true`

### curl
```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email":"user@example.com",
    "password":"StrongPass123",
    "role":"model"
  }'
```

### Пример ответа
```json
{
  "success": true,
  "data": {
    "message": "Registered. Email code sent.",
    "data": {
      "user": {
        "id": "6e5c8cf8-5d76-4d39-8ec2-4d8a98d30f76",
        "email": "user@example.com",
        "role": "model",
        "email_verified": false,
        "is_verified": false,
        "created_at": "2026-02-12T00:00:00Z"
      },
      "verification_sent": true
    }
  }
}
```

## 2) Verify request (public)
`POST /api/v1/auth/verify/request`

- Возвращает `200`
- Ответ нейтральный (без утечки существования email)

### curl
```bash
curl -X POST http://localhost:8080/api/v1/auth/verify/request \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com"}'
```

### Пример ответа
```json
{
  "success": true,
  "data": {
    "status": "sent"
  }
}
```

## 3) Verify confirm (public)
`POST /api/v1/auth/verify/confirm`

- Возвращает `200`
- В ответе пользователь с `email_verified=true` и `is_verified=true`

### curl
```bash
curl -X POST http://localhost:8080/api/v1/auth/verify/confirm \
  -H "Content-Type: application/json" \
  -d '{
    "email":"user@example.com",
    "code":"123456"
  }'
```

### Пример ответа
```json
{
  "success": true,
  "data": {
    "status": "verified",
    "user": {
      "id": "6e5c8cf8-5d76-4d39-8ec2-4d8a98d30f76",
      "email": "user@example.com",
      "role": "model",
      "email_verified": true,
      "is_verified": true,
      "created_at": "2026-02-12T00:00:00Z"
    }
  }
}
```

## 4) Login до verify
`POST /api/v1/auth/login`

- Если email не подтверждён: `403 EMAIL_NOT_VERIFIED`

### curl
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email":"unverified@example.com",
    "password":"StrongPass123"
  }'
```

### Пример ответа
```json
{
  "success": false,
  "error_code": "EMAIL_NOT_VERIFIED",
  "message": "Email is not verified"
}
```

## 5) Login после verify
`POST /api/v1/auth/login`

- Возвращает `200` + токены
- `refresh_token` — **opaque 64 hex**

### curl
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email":"user@example.com",
    "password":"StrongPass123"
  }'
```

### Пример ответа
```json
{
  "success": true,
  "data": {
    "user": {
      "id": "6e5c8cf8-5d76-4d39-8ec2-4d8a98d30f76",
      "email": "user@example.com",
      "role": "model",
      "email_verified": true,
      "is_verified": true,
      "created_at": "2026-02-12T00:00:00Z"
    },
    "tokens": {
      "access_token": "<jwt>",
      "refresh_token": "4f1e7b4f3d6a7b52f8bcb8d0ef68a0f17db2f7e4f226ec2dfcb6a39b4a9f5b0d",
      "expires_in": 900,
      "token_type": "Bearer"
    }
  }
}
```

## 6) Refresh
`POST /api/v1/auth/refresh`

- Возвращает `200`
- Выполняет ротацию refresh-токена

### curl
```bash
curl -X POST http://localhost:8080/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token":"4f1e7b4f3d6a7b52f8bcb8d0ef68a0f17db2f7e4f226ec2dfcb6a39b4a9f5b0d"
  }'
```

### Пример ответа
```json
{
  "success": true,
  "data": {
    "user": {
      "id": "6e5c8cf8-5d76-4d39-8ec2-4d8a98d30f76",
      "email": "user@example.com",
      "role": "model",
      "email_verified": true,
      "is_verified": true,
      "created_at": "2026-02-12T00:00:00Z"
    },
    "tokens": {
      "access_token": "<jwt>",
      "refresh_token": "9fbab61821df5cb456ff2537f63f947ef0f7e8edee13d81fc3ffad9301c48a1e",
      "expires_in": 900,
      "token_type": "Bearer"
    }
  }
}
```

## 7) Logout
`POST /api/v1/auth/logout`

- Возвращает `204 No Content`

### curl
```bash
curl -X POST http://localhost:8080/api/v1/auth/logout \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token":"9fbab61821df5cb456ff2537f63f947ef0f7e8edee13d81fc3ffad9301c48a1e"
  }'
```
