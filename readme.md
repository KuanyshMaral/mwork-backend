# MWork API Documentation

> **Complete REST API for Kazakhstan's premier casting and modeling platform**

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![API Version](https://img.shields.io/badge/API-v1.0-blue)](https://api.mwork.kz)
[![License](https://img.shields.io/badge/License-Proprietary-red)](LICENSE)
[![Status](https://img.shields.io/badge/Status-Production-success)](https://mwork.kz)

---

## 📚 Table of Contents

- [Overview](#-overview)
- [Quick Start](#-quick-start)
- [Authentication](#-authentication)
- [API Reference](#-api-reference)
- [Subscription Plans](#-subscription-plans)
- [Error Handling](#-error-handling)
- [Rate Limiting](#-rate-limiting)
- [Webhooks](#-webhooks)
- [SDKs & Tools](#-sdks--tools)
- [Support](#-support)

---

## 🎯 Overview

**MWork** is a comprehensive casting platform connecting **models** with **employers** (brands, agencies, photographers) across Kazakhstan. The API provides full access to:

- 👤 **User Management**: Registration, authentication, profiles
- 🎬 **Casting System**: Job postings, applications, responses
- 💬 **Real-time Chat**: WebSocket-based messaging
- 📸 **Media Management**: Photo uploads, portfolio management
- 💳 **Subscriptions**: Freemium model with Kaspi payment integration
- 🔔 **Notifications**: Push notifications, email alerts
- 🏢 **Organizations**: Agency and company management

### Key Features

- ✅ **RESTful API** with consistent JSON responses
- ✅ **JWT Authentication** with refresh token support
- ✅ **WebSocket Support** for real-time features
- ✅ **File Uploads** to Cloudflare R2 storage
- ✅ **Payment Integration** with Kaspi
- ✅ **Multi-role System** (model, employer, agency, admin)
- ✅ **Subscription Limits** enforcement
- ✅ **Email Verification** workflow

### Technical Stack

```
Backend:     Go 1.21+ with Chi Router
Database:    PostgreSQL 14+
Cache:       Redis 7+
Storage:     Cloudflare R2 (S3-compatible)
Payments:    Kaspi API
Email:       Resend API
WebSocket:   Gorilla WebSocket
```

---

## 🚀 Quick Start

### 1. Base URL

All API requests should be made to:

```
Production:  https://api.mwork.kz/api/v1
Development: http://localhost:8080/api/v1
```

### 2. Register a User

```bash
curl -X POST https://api.mwork.kz/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "model@example.com",
    "password": "SecurePass123!",
    "role": "model"
  }'
```

**Response:**
```json
{
  "status": "success",
  "data": {
    "user": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "email": "model@example.com",
      "role": "model",
      "email_verified": false,
      "created_at": "2026-02-02T10:00:00Z"
    }
  }
}
```

### 3. Login

```bash
curl -X POST https://api.mwork.kz/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "model@example.com",
    "password": "SecurePass123!"
  }'
```

**Response:**
```json
{
  "status": "success",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_in": 900,
    "user": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "email": "model@example.com",
      "role": "model"
    }
  }
}
```

### 4. Make Authenticated Request

```bash
curl -X GET https://api.mwork.kz/api/v1/auth/me \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

---

## 🔐 Authentication

MWork API uses **JWT (JSON Web Tokens)** for authentication.

### Authentication Flow

```
1. Register → Receive user account
2. Verify Email → Click verification link (optional but recommended)
3. Login → Receive access_token + refresh_token
4. Use Token → Include in Authorization header
5. Refresh Token → Get new access_token when expired
```

### Token Types

| Token Type | Validity | Purpose |
|------------|----------|---------|
| **Access Token** | 15 minutes | API requests |
| **Refresh Token** | 7 days | Renew access token |

### Header Format

```http
Authorization: Bearer YOUR_ACCESS_TOKEN
```

### Refreshing Tokens

```bash
curl -X POST https://api.mwork.kz/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "YOUR_REFRESH_TOKEN"
  }'
```

### User Roles

- **`model`**: Individual models, can apply to castings
- **`employer`**: Brands, photographers, create castings
- **`agency`**: Modeling agencies, manage multiple models
- **`admin`**: Platform administrators

---

## 📖 API Reference

### Core Resources

| Resource | Description | Public Access |
|----------|-------------|---------------|
| [Authentication](#authentication-endpoints) | User auth & tokens | Yes |
| [Profiles](#profiles-endpoints) | Model & employer profiles | Partial |
| [Castings](#castings-endpoints) | Job postings | Partial |
| [Responses](#responses-endpoints) | Casting applications | No |
| [Chat](#chat-endpoints) | Real-time messaging | No |
| [Photos](#photos-endpoints) | Portfolio management | Partial |
| [Subscriptions](#subscriptions-endpoints) | Plans & billing | Partial |
| [Notifications](#notifications-endpoints) | Push & email alerts | No |

### Authentication Endpoints

#### POST /auth/register
Create a new user account.

**Request:**
```json
{
  "email": "user@example.com",
  "password": "SecurePass123!",
  "role": "model" // model, employer, or agency
}
```

**Response: 201 Created**
```json
{
  "status": "success",
  "data": {
    "user": {
      "id": "uuid",
      "email": "user@example.com",
      "role": "model",
      "email_verified": false
    }
  }
}
```

**Validation:**
- Email must be valid and unique
- Password minimum 8 characters
- Role must be one of: model, employer, agency

---

#### POST /auth/login
Authenticate and receive tokens.

**Request:**
```json
{
  "email": "user@example.com",
  "password": "SecurePass123!"
}
```

**Response: 200 OK**
```json
{
  "status": "success",
  "data": {
    "access_token": "eyJhbGc...",
    "refresh_token": "eyJhbGc...",
    "expires_in": 900,
    "user": { ... }
  }
}
```

---

#### POST /auth/refresh
Refresh expired access token.

**Request:**
```json
{
  "refresh_token": "eyJhbGc..."
}
```

**Response: 200 OK**
```json
{
  "status": "success",
  "data": {
    "access_token": "eyJhbGc...",
    "expires_in": 900
  }
}
```

---

#### POST /auth/logout
🔒 Invalidate current tokens.

**Headers:** `Authorization: Bearer {token}`

**Response: 200 OK**
```json
{
  "status": "success",
  "message": "Logged out successfully"
}
```

---

#### GET /auth/me
🔒 Get current user information.

**Headers:** `Authorization: Bearer {token}`

**Response: 200 OK**
```json
{
  "status": "success",
  "data": {
    "user": {
      "id": "uuid",
      "email": "user@example.com",
      "role": "model",
      "email_verified": true,
      "subscription": {
        "plan": "PRO",
        "expires_at": "2026-03-02T00:00:00Z"
      }
    }
  }
}
```

---

### Profiles Endpoints

#### GET /profiles/models
List all model profiles with filtering.

**Query Parameters:**
```
limit        int       Items per page (default: 20, max: 100)
offset       int       Pagination offset (default: 0)
gender       string    Filter by gender (male/female/other)
min_age      int       Minimum age
max_age      int       Maximum age
min_height   int       Minimum height in cm
max_height   int       Maximum height in cm
city         string    Filter by city
category     string    Model category (fashion/commercial/editorial/runway)
experience   string    Experience level (beginner/intermediate/experienced)
search       string    Search in name/bio
```

**Response: 200 OK**
```json
{
  "status": "success",
  "data": {
    "models": [
      {
        "id": "uuid",
        "user_id": "uuid",
        "name": "Aisha Nurzhanova",
        "age": 24,
        "gender": "female",
        "height": 175,
        "city": "Almaty",
        "categories": ["fashion", "editorial"],
        "experience": "experienced",
        "main_photo": "https://cdn.mwork.kz/photos/...",
        "is_promoted": false,
        "rating": 4.8,
        "reviews_count": 12
      }
    ],
    "pagination": {
      "total": 156,
      "limit": 20,
      "offset": 0,
      "has_more": true
    }
  }
}
```

---

#### GET /profiles/models/promoted
Get featured/promoted models.

**Query Parameters:**
```
limit    int    Number of promoted models (default: 10, max: 50)
```

**Response: 200 OK**
```json
{
  "status": "success",
  "data": {
    "models": [ ... ],
    "promoted_until": "2026-02-10T00:00:00Z"
  }
}
```

---

#### GET /profiles/models/:id
Get detailed model profile.

**Response: 200 OK**
```json
{
  "status": "success",
  "data": {
    "model": {
      "id": "uuid",
      "user_id": "uuid",
      "name": "Aisha Nurzhanova",
      "bio": "Professional fashion model...",
      "age": 24,
      "gender": "female",
      "height": 175,
      "weight": 58,
      "measurements": {
        "bust": 86,
        "waist": 62,
        "hips": 90
      },
      "shoe_size": 38,
      "clothing_size": "S",
      "hair_color": "brown",
      "eye_color": "brown",
      "skin_tone": "fair",
      "city": "Almaty",
      "country": "Kazakhstan",
      "languages": ["Kazakh", "Russian", "English"],
      "categories": ["fashion", "commercial", "editorial"],
      "experience": "experienced",
      "can_travel": true,
      "photos": [
        {
          "id": "uuid",
          "url": "https://cdn.mwork.kz/photos/...",
          "is_main": true,
          "order": 0
        }
      ],
      "work_experience": [
        {
          "company": "Fashion House Almaty",
          "position": "Fashion Model",
          "start_date": "2020-01-01",
          "end_date": "2023-12-31"
        }
      ],
      "rating": 4.8,
      "reviews_count": 12,
      "completeness": 95
    }
  }
}
```

---

#### POST /profiles/models
🔒 Create model profile (requires model role).

**Request:**
```json
{
  "name": "Aisha Nurzhanova",
  "bio": "Professional fashion model with 5 years of experience",
  "age": 24,
  "gender": "female",
  "height": 175,
  "weight": 58,
  "bust": 86,
  "waist": 62,
  "hips": 90,
  "shoe_size": 38,
  "clothing_size": "S",
  "hair_color": "brown",
  "eye_color": "brown",
  "skin_tone": "fair",
  "city": "Almaty",
  "country": "Kazakhstan",
  "languages": ["Kazakh", "Russian", "English"],
  "categories": ["fashion", "commercial"],
  "experience": "experienced",
  "can_travel": true
}
```

**Response: 201 Created**

---

#### PUT /profiles/models/:id
🔒 Update model profile (owner only).

**Request:**
```json
{
  "bio": "Updated bio...",
  "height": 176,
  "categories": ["fashion", "editorial", "runway"]
}
```

**Response: 200 OK**

---

#### POST /profiles/employers
🔒 Create employer profile (requires employer/agency role).

**Request:**
```json
{
  "company_name": "Fashion House Almaty",
  "description": "Leading fashion agency in Kazakhstan",
  "website": "https://fashionhouse.kz",
  "contact_person": "Yerlan Akhmetov",
  "phone": "+77771234567",
  "city": "Almaty",
  "country": "Kazakhstan"
}
```

**Response: 201 Created**

---

#### GET /profiles/me
🔒 Get current user's profile.

**Response: 200 OK**

---

### Castings Endpoints

#### GET /castings
List all castings with filters.

**Query Parameters:**
```
limit      int       Items per page (default: 20)
offset     int       Pagination offset
status     string    Filter by status (active/draft/completed/cancelled)
city       string    Filter by city
category   string    Casting category
gender     string    Required gender
min_age    int       Minimum age requirement
max_age    int       Maximum age requirement
search     string    Search in title/description
```

**Response: 200 OK**
```json
{
  "status": "success",
  "data": {
    "castings": [
      {
        "id": "uuid",
        "title": "Fashion Editorial Shoot",
        "description": "Looking for female models...",
        "category": "fashion",
        "city": "Almaty",
        "shoot_date": "2026-03-15T10:00:00Z",
        "pay_min": 50000,
        "pay_max": 100000,
        "status": "active",
        "requirements": {
          "gender": "female",
          "min_age": 18,
          "max_age": 30,
          "min_height": 170
        },
        "creator": {
          "id": "uuid",
          "company_name": "Fashion House",
          "avatar": "https://..."
        },
        "responses_count": 24,
        "created_at": "2026-02-01T00:00:00Z"
      }
    ],
    "pagination": { ... }
  }
}
```

---

#### GET /castings/:id
Get detailed casting information.

**Response: 200 OK**

---

#### POST /castings
🔒 Create new casting (requires employer/agency role).

**Request:**
```json
{
  "title": "Fashion Editorial Shoot",
  "description": "Looking for female models for editorial fashion shoot...",
  "category": "fashion",
  "city": "Almaty",
  "address": "Dostyk Avenue 123",
  "shoot_date": "2026-03-15T10:00:00Z",
  "duration_days": 1,
  "pay_min": 50000,
  "pay_max": 100000,
  "requirements": {
    "gender": "female",
    "min_age": 18,
    "max_age": 30,
    "min_height": 170,
    "max_height": 185,
    "experience": "experienced"
  }
}
```

**Response: 201 Created**

---

#### PUT /castings/:id
🔒 Update casting (owner only).

---

#### PATCH /castings/:id/status
🔒 Update casting status.

**Request:**
```json
{
  "status": "active" // draft/active/completed/cancelled
}
```

---

#### DELETE /castings/:id
🔒 Delete casting (owner only).

**Response: 204 No Content**

---

#### GET /castings/my
🔒 Get my castings (employer/agency).

---

#### GET /castings/saved
🔒 Get saved/favorite castings.

---

#### POST /castings/:id/save
🔒 Add casting to favorites.

---

#### DELETE /castings/:id/save
🔒 Remove from favorites.

---

### Responses Endpoints

#### POST /castings/:id/responses
🔒📊 Apply to casting (models only, subscription limit applies).

**Request:**
```json
{
  "message": "I'm interested in this casting. I have 5 years of experience...",
  "portfolio_links": [
    "https://instagram.com/model",
    "https://portfolio.example.com"
  ]
}
```

**Response: 201 Created**

---

#### GET /castings/:id/responses
🔒 Get all responses for a casting (casting owner only).

**Response: 200 OK**
```json
{
  "status": "success",
  "data": {
    "responses": [
      {
        "id": "uuid",
        "applicant": {
          "id": "uuid",
          "name": "Aisha Nurzhanova",
          "photo": "https://...",
          "age": 24,
          "height": 175,
          "city": "Almaty",
          "rating": 4.8
        },
        "message": "I'm interested...",
        "status": "pending",
        "applied_at": "2026-02-02T10:00:00Z"
      }
    ]
  }
}
```

---

#### GET /responses/my
🔒 Get my applications (models).

---

#### PATCH /responses/:id/status
🔒 Update response status (casting owner only).

**Request:**
```json
{
  "status": "accepted" // pending/accepted/rejected
}
```

---

### Chat Endpoints

#### WebSocket: /ws?token={jwt}
🔒 Connect to WebSocket for real-time messaging.

**Connection:**
```javascript
const ws = new WebSocket('wss://api.mwork.kz/ws?token=YOUR_JWT');

ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  console.log('New message:', message);
};

ws.send(JSON.stringify({
  type: 'message',
  room_id: 'uuid',
  content: 'Hello!'
}));
```

---

#### POST /chat/rooms
🔒 Create chat room.

**Request:**
```json
{
  "participant_id": "uuid",
  "casting_id": "uuid" // optional
}
```

---

#### GET /chat/rooms
🔒📊 Get all chat rooms (subscription limit).

---

#### GET /chat/rooms/:id/messages
🔒 Get message history.

**Query Parameters:**
```
limit     int    Messages per page (default: 50)
before    uuid   Get messages before this ID (pagination)
```

---

#### POST /chat/rooms/:id/messages
🔒📊 Send message (subscription limit).

**Request:**
```json
{
  "content": "Hello! I'm interested in discussing..."
}
```

---

#### POST /chat/rooms/:id/read
🔒 Mark all messages as read.

---

#### GET /chat/unread
🔒 Get unread message count.

---

### Photos Endpoints

#### POST /photos
🔒📊 Upload portfolio photo (subscription limit applies).

**Request:** `multipart/form-data`
```
file: [binary]
is_main: false
order: 1
```

**Response: 201 Created**
```json
{
  "status": "success",
  "data": {
    "photo": {
      "id": "uuid",
      "url": "https://cdn.mwork.kz/photos/...",
      "thumbnail_url": "https://cdn.mwork.kz/thumbnails/...",
      "is_main": false,
      "order": 1
    }
  }
}
```

---

#### GET /photos
Get profile photos.

**Query Parameters:**
```
profile_id    uuid    Required. Profile ID to fetch photos for
```

---

#### PUT /photos/:id
🔒 Update photo metadata.

**Request:**
```json
{
  "is_main": true,
  "order": 0
}
```

---

#### DELETE /photos/:id
🔒 Delete photo.

---

#### POST /photos/reorder
🔒 Reorder photos.

**Request:**
```json
{
  "photo_ids": ["uuid1", "uuid2", "uuid3"]
}
```

---

### Subscriptions Endpoints

#### GET /subscriptions/plans
Get all available subscription plans.

**Response: 200 OK**
```json
{
  "status": "success",
  "data": {
    "plans": [
      {
        "name": "FREE",
        "price_monthly": 0,
        "price_yearly": 0,
        "features": {
          "max_photos": 5,
          "max_responses_per_month": 3,
          "chat_access": false,
          "profile_promotion": false
        }
      },
      {
        "name": "PRO",
        "price_monthly": 5000,
        "price_yearly": 50000,
        "features": {
          "max_photos": 20,
          "max_responses_per_month": 30,
          "chat_access": true,
          "profile_promotion": false
        }
      },
      {
        "name": "BUSINESS",
        "price_monthly": 15000,
        "price_yearly": 150000,
        "features": {
          "max_photos": -1,
          "max_responses_per_month": -1,
          "chat_access": true,
          "profile_promotion": true
        }
      }
    ]
  }
}
```

---

#### GET /subscriptions/my
🔒 Get current subscription status.

**Response: 200 OK**
```json
{
  "status": "success",
  "data": {
    "subscription": {
      "plan": "PRO",
      "status": "active",
      "started_at": "2026-01-01T00:00:00Z",
      "expires_at": "2026-03-01T00:00:00Z",
      "auto_renew": true,
      "usage": {
        "photos_uploaded": 12,
        "photos_limit": 20,
        "responses_sent": 8,
        "responses_limit": 30
      }
    }
  }
}
```

---

#### POST /subscriptions/subscribe
🔒 Subscribe to a plan.

**Request:**
```json
{
  "plan_name": "PRO",
  "billing_period": "monthly" // monthly or yearly
}
```

**Response: 200 OK**
```json
{
  "status": "success",
  "data": {
    "payment_url": "https://kaspi.kz/pay/...",
    "order_id": "order-123",
    "amount": 5000
  }
}
```

---

#### POST /subscriptions/cancel
🔒 Cancel subscription.

---

### Notifications Endpoints

#### GET /notifications
🔒 List user notifications.

**Query Parameters:**
```
limit          int     Items per page (default: 20)
offset         int     Pagination offset
unread_only    bool    Show only unread (default: false)
```

**Response: 200 OK**
```json
{
  "status": "success",
  "data": {
    "notifications": [
      {
        "id": "uuid",
        "type": "new_response",
        "title": "New Response to Your Casting",
        "message": "Aisha Nurzhanova applied to Fashion Editorial Shoot",
        "data": {
          "casting_id": "uuid",
          "response_id": "uuid"
        },
        "is_read": false,
        "created_at": "2026-02-02T10:00:00Z"
      }
    ]
  }
}
```

---

#### POST /notifications/:id/read
🔒 Mark notification as read.

---

#### POST /notifications/read-all
🔒 Mark all notifications as read.

---

#### GET /notifications/preferences
🔒 Get notification preferences.

---

#### PUT /notifications/preferences
🔒 Update notification preferences.

**Request:**
```json
{
  "new_casting": true,
  "new_response": true,
  "response_status_change": true,
  "new_message": true,
  "casting_reminder": false
}
```

---

## 💎 Subscription Plans

MWork operates on a freemium model with three subscription tiers:

| Feature | FREE | PRO | BUSINESS |
|---------|------|-----|----------|
| **Price (Monthly)** | ₸0 | ₸5,000 | ₸15,000 |
| **Price (Yearly)** | ₸0 | ₸50,000 | ₸150,000 |
| **Portfolio Photos** | 5 | 20 | Unlimited |
| **Casting Responses/Month** | 3 | 30 | Unlimited |
| **Chat Access** | ❌ | ✅ | ✅ |
| **Profile Promotion** | ❌ | ❌ | ✅ |
| **Priority Support** | ❌ | ❌ | ✅ |
| **Analytics Dashboard** | ❌ | Basic | Advanced |

### Payment Methods

- **Kaspi Pay**: Kazakhstan's leading payment system
- Auto-renewal supported for all plans
- Cancel anytime (access until period end)

---

## ⚠️ Error Handling

### Error Response Format

```json
{
  "status": "error",
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid request parameters",
    "details": {
      "email": ["Email is required", "Email must be valid"],
      "age": ["Age must be between 18 and 99"]
    }
  }
}
```

### HTTP Status Codes

| Code | Description |
|------|-------------|
| `200` | Success |
| `201` | Created |
| `204` | No Content (successful deletion) |
| `400` | Bad Request (validation error) |
| `401` | Unauthorized (invalid/missing token) |
| `403` | Forbidden (subscription limit / insufficient permissions) |
| `404` | Not Found |
| `409` | Conflict (duplicate resource) |
| `422` | Unprocessable Entity |
| `429` | Too Many Requests (rate limit exceeded) |
| `500` | Internal Server Error |

### Common Error Codes

| Code | Description |
|------|-------------|
| `AUTH_REQUIRED` | Authentication required |
| `INVALID_TOKEN` | JWT token is invalid or expired |
| `VALIDATION_ERROR` | Request validation failed |
| `NOT_FOUND` | Resource not found |
| `PERMISSION_DENIED` | Insufficient permissions |
| `SUBSCRIPTION_LIMIT` | Subscription limit reached |
| `DUPLICATE_RESOURCE` | Resource already exists |
| `RATE_LIMIT_EXCEEDED` | Too many requests |

---

## ⏱️ Rate Limiting

To ensure fair usage and system stability, the API implements rate limiting:

### Limits

| Endpoint Category | Limit | Window |
|-------------------|-------|--------|
| Authentication | 5 requests | per minute |
| API Endpoints (authenticated) | 100 requests | per minute |
| File Uploads | 10 requests | per minute |
| Webhooks | 1000 requests | per minute |

### Rate Limit Headers

```http
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1707739200
```

### Exceeded Rate Limit Response

**429 Too Many Requests**
```json
{
  "status": "error",
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "Rate limit exceeded. Please try again later.",
    "retry_after": 60
  }
}
```

---

## 🔗 Webhooks

### Kaspi Payment Webhook

**Endpoint:** `POST /webhooks/kaspi`

Kaspi sends payment status updates to this endpoint.

**Payload:**
```json
{
  "order_id": "order-123",
  "status": "completed",
  "amount": 5000,
  "transaction_id": "txn-456",
  "timestamp": "2026-02-02T10:00:00Z"
}
```

**Statuses:**
- `pending`: Payment initiated
- `completed`: Payment successful
- `failed`: Payment failed
- `cancelled`: Payment cancelled

---

## 🛠️ SDKs & Tools

### Postman Collection

Import our comprehensive Postman collection for easy API testing:

**Collection URL:** [Download MWork_API_Collection_v2.json](./MWork_API_Collection_v2.postman_collection.json)

**Features:**
- All 115+ endpoints documented
- Pre-configured authentication
- Example requests and responses
- Environment variables for easy switching
- Test scripts for common workflows

### cURL Examples

**List Models:**
```bash
curl -X GET "https://api.mwork.kz/api/v1/profiles/models?limit=10&city=Almaty" \
  -H "Accept: application/json"
```

**Create Casting:**
```bash
curl -X POST "https://api.mwork.kz/api/v1/castings" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Fashion Shoot",
    "description": "Looking for models...",
    "city": "Almaty",
    "shoot_date": "2026-03-15T10:00:00Z",
    "pay_min": 50000,
    "pay_max": 100000
  }'
```

**Upload Photo:**
```bash
curl -X POST "https://api.mwork.kz/api/v1/photos" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -F "file=@/path/to/photo.jpg" \
  -F "is_main=false" \
  -F "order=1"
```

---

## 📞 Support

### Documentation

- **API Reference**: [https://docs.mwork.kz](https://docs.mwork.kz)
- **Changelog**: [CHANGELOG.md](./CHANGELOG.md)
- **Migration Guide**: [MIGRATION.md](./MIGRATION.md)

### Contact

- **Technical Support**: tech@mwork.kz
- **Business Inquiries**: business@mwork.kz
- **Emergency Contact**: +7 (777) 123-4567

### Community

- **GitHub Issues**: [github.com/mwork/api/issues](https://github.com/mwork/api/issues)
- **Telegram**: [@mwork_dev](https://t.me/mwork_dev)
- **Stack Overflow**: Tag your questions with `mwork-api`

---

## 📄 License

Copyright © 2026 MWork. All rights reserved.

This API documentation is proprietary and confidential. Unauthorized use, reproduction, or distribution is strictly prohibited.

---

## 🔄 Version History

| Version | Date | Changes |
|---------|------|---------|
| v1.0.0 | 2026-02-02 | Initial production release |
| v0.9.0 | 2025-12-15 | Beta release with core features |
| v0.5.0 | 2025-10-01 | Alpha release for testing |

---

**Last Updated:** February 02, 2026  
**API Version:** 1.0.0  
**Documentation Version:** 2.0.0
