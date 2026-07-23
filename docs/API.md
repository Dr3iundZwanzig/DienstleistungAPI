# DienstleistungAPI Documentation

## Overview

DienstleistungAPI is a RESTful API for managing service appointments, employees, availability, and services. It provides secure authentication using JWT access tokens, refresh token rotation, and server-side session invalidation.

## Base URL

```
http://localhost:{PORT}/api
```

Replace `{PORT}` with the configured port (from environment variable `PORT`).

## Authentication

The API uses JWT (JSON Web Token) based authentication with refresh token rotation for sensitive endpoints.

### Token Types

- **Access Token**: Short-lived JWT for authenticating API requests (default TTL: 24h)
- **Refresh Token**: Longer-lived token stored in the database for obtaining new access tokens (default TTL: 168h / 7 days)

Access tokens include a `session_version` claim. Protected endpoints validate this against the user's current `session_version` in the database. If the values differ, the token is rejected.

### Bearer Token Format

Include the token in the `Authorization` header:

```
Authorization: Bearer {token}
```

## Rate Limiting

The API implements rate limiting per client IP, configurable in the .env file:

- **Login endpoint**: 10 requests/minute (configurable via `LOGIN_RATE_LIMIT_PER_MINUTE`)
- **Failed login attempts**: 5 failed attempts/minute (configurable via `LOGIN_FAILED_RATE_LIMIT_PER_MINUTE`)
- **Refresh endpoint**: 30 requests/minute (configurable via `REFRESH_RATE_LIMIT_PER_MINUTE`)

Rate limit violations return `429 Too Many Requests`.

## API Endpoints

### Authentication

#### Login

Create a new session and obtain access and refresh tokens.

```
POST /api/login
```

**Rate Limited**: 10 requests/minute per IP

**Request Body**:
```json
{
  "email": "user@example.com",
  "password": "password123"
}
```

**Response** (200 OK):
```json
{
  "id": "uuid",
  "email": "user@example.com",
  "role": "customer",
  "session_version": 1,
  "created_at": "2024-01-15T10:30:00Z",
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "refresh_token_plaintext"
}
```

**Errors**:
- `401 Unauthorized`: Incorrect email or password
- `429 Too Many Requests`: Rate limit exceeded

---

#### Refresh Token

Obtain a new access token using a valid refresh token. Automatically rotates the refresh token.

```
POST /api/refresh
Authorization: Bearer {refresh_token}
```

**Rate Limited**: 30 requests/minute per IP

**Response** (200 OK):
```json
{
  "token": "new_eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "new_refresh_token_hash"
}
```

**Notes**:
- Old refresh token is invalidated
- Client must store and use the new refresh token
- Returns new access token with shorter TTL (default: 1h, configurable via `REFRESH_ACCESS_TOKEN_TTL`)

**Errors**:
- `400 Bad Request`: Token not found in header
- `401 Unauthorized`: Invalid or expired refresh token
- `429 Too Many Requests`: Rate limit exceeded

---

#### Revoke Session

Revoke one refresh token.

```
POST /api/revoke
Authorization: Bearer {refresh_token}
```

**Response** (204 No Content):
- No response body

**Notes**:
- This revokes only the provided refresh token
- Existing access tokens remain valid until they expire or sessions are invalidated server-side

**Errors**:
- `400 Bad Request`: Token not found in header
- `500 Internal Server Error`: Server error during revocation

---

#### Logout All Sessions

Invalidate all active sessions for the authenticated user in one action.

```
POST /api/logout-all
Authorization: Bearer {access_token}
```

**Response** (204 No Content):
- No response body

**Notes**:
- Increments the user's `session_version` so previously issued access tokens become invalid
- Revokes all non-revoked refresh tokens for the user
- Client should clear local tokens immediately after calling this endpoint

**Errors**:
- `401 Unauthorized`: Missing or invalid access token
- `500 Internal Server Error`: Server error during session invalidation

---

### Users

#### Create User

Register a new user account.

```
POST /api/users
```

**Request Body**:
```json
{
  "email": "newuser@example.com",
  "password": "securepassword"
}
```

**Response** (201 Created):
```json
{
  "id": "uuid",
  "email": "newuser@example.com",
  "role": "customer",
  "session_version": 1,
  "created_at": "2024-01-15T10:30:00Z"
}
```

**Errors**:
- `400 Bad Request`: Email or password missing
- `500 Internal Server Error`: User already exists or database error

---

### Appointments

#### List Appointments

Retrieve all appointments for the authenticated user.

```
GET /api/appointments
Authorization: Bearer {access_token}
```

**Response** (200 OK):
```json
{
  "data": [
    {
      "id": "appointment-uuid",
      "user_id": "user-uuid",
      "employee_id": "employee-id",
      "date": "2024-02-20",
      "start_time": "10:00",
      "end_time": "11:00",
      "services": [
        {
          "id": "service-id",
          "name": "Haarschnitt",
          "description": "Waschen, Schneiden, Föhnen",
          "duration_minutes": 45,
          "price": 39.90,
          "currency": "EUR"
        }
      ],
      "total_duration_minutes": 45,
      "total_price": 39.90,
      "status": "confirmed",
      "created_at": "2024-01-15T10:30:00Z"
    }
  ]
}
```

**Errors**:
- `401 Unauthorized`: Missing or invalid token

---

#### Create Appointment

Book a new appointment. Automatically selects an available employee if not specified.

```
POST /api/appointments
Authorization: Bearer {access_token}
```

**Request Body**:
```json
{
  "date": "2024-02-20",
  "start_time": "10:00",
  "end_time": "11:00",
  "service_ids": ["srv_001", "srv_002"],
  "employee_id": "emp-uuid",
  "no_preference": false
}
```

**Parameters**:
- `date` (required): Appointment date in YYYY-MM-DD format
- `start_time` (required): Start time in HH:MM format
- `end_time` (required): End time in HH:MM format
- `service_ids` (required): Array of service IDs to book
- `employee_id` (optional): Specific employee UUID; if omitted and `no_preference` is false, system finds best match
- `no_preference` (optional): If true, allows any available employee

**Response** (201 Created):
```json
{
  "id": "appointment-uuid",
  "user_id": "user-uuid",
  "employee_id": "employee-id",
  "date": "2024-02-20",
  "start_time": "10:00",
  "end_time": "11:00",
  "services": [
    {
      "id": "service-id",
      "name": "Haarschnitt",
      "price": 39.90
    }
  ],
  "total_duration_minutes": 45,
  "total_price": 39.90,
  "status": "confirmed",
  "created_at": "2024-01-15T10:30:00Z"
}
```

**Errors**:
- `400 Bad Request`: Invalid request body or missing required fields
- `401 Unauthorized`: Missing or invalid token
- `409 Conflict`: Appointment slot not available

---

#### Update Appointment

Modify an existing appointment.

```
PUT /api/appointments/{id}
Authorization: Bearer {access_token}
```

**Parameters**:
- `id` (path parameter): Appointment UUID

**Request Body**:
```json
{
  "date": "2024-02-21",
  "start_time": "14:00",
  "end_time": "15:00",
  "service_ids": ["srv_001"],
  "employee_id": "emp-uuid"
}
```

**Response** (200 OK):
```json
{
  "id": "appointment-uuid",
  "user_id": "user-uuid",
  "employee_id": "employee-id",
  "date": "2024-02-21",
  "start_time": "14:00",
  "end_time": "15:00",
  "services": [...],
  "total_duration_minutes": 45,
  "total_price": 39.90,
  "status": "confirmed",
  "created_at": "2024-01-15T10:30:00Z"
}
```

**Errors**:
- `401 Unauthorized`: Missing or invalid token
- `404 Not Found`: Appointment not found
- `409 Conflict`: New slot not available

---

#### Cancel Appointment

Cancel a single appointment.

```
DELETE /api/appointments/{id}
Authorization: Bearer {access_token}
```

**Parameters**:
- `id` (path parameter): Appointment UUID

**Response** (204 No Content):
- No response body

**Errors**:
- `401 Unauthorized`: Missing or invalid token
- `404 Not Found`: Appointment not found

---

#### Cancel All Appointments

Cancel all appointments for the authenticated user. **Test endpoint only.**

```
DELETE /api/appointments/delete
Authorization: Bearer {access_token}
```

**Response** (204 No Content):
- No response body

**Notes**:
- This endpoint is intended for testing purposes, should be removed later
- Cancels all appointments belonging to the user

**Errors**:
- `401 Unauthorized`: Missing or invalid token

---

### Availability

#### Get Employee Availability

Retrieve available time slots for an employee.

```
GET /api/availability?employee_id={employee_id}
```

**Query Parameters**:
- `employee_id` (required): Employee ID

**Response** (200 OK):
```json
{
  "employee_id": "emp-001",
  "dates": [
    {
      "date": "2024-02-20",
      "time_slots": [
        {
          "start_time": "09:00",
          "end_time": "10:00",
          "is_available": true
        },
        {
          "start_time": "10:00",
          "end_time": "11:00",
          "is_available": false
        }
      ]
    }
  ]
}
```

**Notes**:
- If employee has no availability record, system automatically seeds with default availability when seeding employees

**Errors**:
- `400 Bad Request`: Missing employee_id parameter
- `500 Internal Server Error`: Database error

---

#### Set Employee Availability

Create or update availability schedule for an employee for later implementation of employee/admin scopes.

```
POST /api/availability
Authorization: Bearer {access_token}
```

**Authentication**: Requires staff or admin role

**Request Body**:
```json
{
  "employee_id": "emp-001",
  "dates": [
    {
      "date": "2024-02-20",
      "time_slots": [
        {
          "start_time": "09:00",
          "end_time": "10:00",
          "is_available": true
        },
        {
          "start_time": "10:00",
          "end_time": "17:00",
          "is_available": true
        }
      ]
    }
  ]
}
```

**Response** (201 Created):
```json
{
  "employee_id": "emp-001",
  "dates": [...],
  "updated_at": "2024-01-15T10:30:00Z"
}
```

**Errors**:
- `401 Unauthorized`: Missing token or insufficient permissions (requires staff/admin)
- `400 Bad Request`: Invalid request body

---

### Employees

#### List Employees

Retrieve all active employees.

```
GET /api/employees
```

**Response** (200 OK):
```json
{
  "data": [
    {
      "id": "emp-001",
      "name": "Max Mustermann",
      "title": "Senior Stylist",
      "specialties": ["srv_001", "srv_002", "srv_005"],
      "is_active": true
    },
    {
      "id": "emp-002",
      "name": "Petra Schmidt",
      "title": "Stylist",
      "specialties": ["srv_001", "srv_003"],
      "is_active": true
    }
  ]
}
```

**Errors**:
- `500 Internal Server Error`: Database error

---

#### Resolve Employee

Find the best available employee for a specific service based on criteria.

```
POST /api/employees/resolve
Authorization: Bearer {access_token}
```

**Request Body**:
```json
{
  "date": "2024-02-20",
  "start_time": "10:00",
  "end_time": "11:00",
  "service_ids": ["srv_001"]
}
```

**Response** (200 OK):
```json
{
  "employee_id": "emp-001",
  "employee_name": "Max Mustermann",
  "title": "Senior Stylist",
  "availability_status": "available"
}
```

**Errors**:
- `400 Bad Request`: Invalid request or no employees available
- `401 Unauthorized`: Missing or invalid token

---

### Services

#### Get Services Tree

Retrieve the hierarchical structure of all available services.

```
GET /api/services/tree
```

**Response** (200 OK):
```json
{
  "data": [
    {
      "id": "cat_01",
      "name": "Friseur",
      "is_active": true,
      "children": [
        {
          "id": "sub_01",
          "name": "Herren",
          "is_active": true,
          "children": [
            {
              "id": "srv_001",
              "name": "Haarschnitt",
              "description": "Waschen, Schneiden, Föhnen",
              "duration_minutes": 45,
              "price": 39.90,
              "currency": "EUR",
              "is_active": true,
              "children": []
            }
          ]
        }
      ]
    }
  ]
}
```

**Errors**:
- `500 Internal Server Error`: Database error

---

### Testing (Development Only)

#### Reset and Seed Database

Reset the database and populate with default seed data.

```
POST /api/test/reset-and-seed
Authorization: Bearer {access_token}
```

**Usage with curl**:
- Windows: curl.exe -X POST "http://localhost:{PORT}/api/test/reset-and-seed"
- Linux: curl -X POST "http://localhost:{PORT}/api/test/reset-and-seed"

**Authentication**: Requires staff or admin role

**Response** (204 No Content):
- No response body

**Notes**:
- **WARNING**: This endpoint deletes all data and repopulates with defaults
- Only available in development/test environments
- Requires staff or admin role

**Errors**:
- `401 Unauthorized`: Missing token or insufficient permissions
- `500 Internal Server Error`: Database error

---

## Error Responses

All error responses follow this format:

```json
{
  "error": "Error message describing what went wrong"
}
```

### Common HTTP Status Codes

- `200 OK`: Request successful
- `201 Created`: Resource created successfully
- `204 No Content`: Request successful, no content to return
- `400 Bad Request`: Invalid request parameters or body
- `401 Unauthorized`: Missing, invalid, or expired authentication token
- `404 Not Found`: Requested resource not found
- `409 Conflict`: Request conflicts with current resource state (e.g., double-booking)
- `429 Too Many Requests`: Rate limit exceeded
- `500 Internal Server Error`: Server error occurred

---

## Environment Variables

### Required

- `DB_PATH`: Database connection string
- `JWT_SECRET`: Secret key for signing JWT tokens
- `PLATFORM`: Platform identifier
- `FILEPATH_ROOT`: Root path for file storage
- `PORT`: HTTP server port

### Optional (with defaults)

- `JWT_ISSUER` (default: `dienstleistung-api`): JWT issuer claim
- `JWT_AUDIENCE` (default: `dienstleistung-api-users`): JWT audience claim
- `ACCESS_TOKEN_TTL` (default: `24h`): Access token lifetime (Go duration format)
- `REFRESH_TOKEN_TTL` (default: `168h`): Refresh token lifetime (Go duration format)
- `REFRESH_ACCESS_TOKEN_TTL` (default: `1h`): Access token lifetime when obtained via refresh (Go duration format)
- `LOGIN_RATE_LIMIT_PER_MINUTE` (default: `10`): Max login requests per minute
- `LOGIN_FAILED_RATE_LIMIT_PER_MINUTE` (default: `5`): Max failed login attempts per minute
- `REFRESH_RATE_LIMIT_PER_MINUTE` (default: `30`): Max refresh requests per minute

### Duration Format Examples

- `24h` = 24 hours
- `15m` = 15 minutes
- `720h` = 30 days (note: Go doesn't support days directly)
- `1h30m` = 1 hour 30 minutes

---

## Authentication Flow

### Initial Login

1. Client sends email and password to `POST /api/login`
2. API returns `access_token` and `refresh_token`
3. Client stores both tokens

### Subsequent Requests

1. Client includes `access_token` in `Authorization: Bearer` header
2. API validates token and processes request
3. If token is valid, request proceeds; if invalid/expired, returns `401 Unauthorized`

### Token Refresh

1. When access token expires, client sends `refresh_token` to `POST /api/refresh`
2. API validates refresh token and returns new `access_token` and new `refresh_token`
3. **Important**: Client must replace the old refresh token with the new one
4. Old refresh token is invalidated

### Logout

1. Client sends `refresh_token` to `POST /api/revoke`
2. Only that refresh token is invalidated

### Logout All Devices/Sessions

1. Client sends `access_token` to `POST /api/logout-all`
2. API increments `session_version` (invalidates older access tokens)
3. API revokes all refresh tokens for that user

---

## Data Types

### Appointment Status

- `confirmed`: Appointment is confirmed and active
- `cancelled`: Appointment has been cancelled
- `completed`: Appointment has been completed

### User Roles

- `user`: Regular user can book appointments
- `staff`: Staff member can manage availability and services
- `admin`: Administrator with full system access

---

## Examples

### Complete Booking Flow

```bash
# 1. Register user
curl.exe -X POST http://localhost:3000/api/users \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"secure123"}'

# 2. Login
curl.exe -X POST http://localhost:3000/api/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"secure123"}'
# Response includes access_token and refresh_token

# 3. Get available services
curl.exe http://localhost:3000/api/services/tree

# 4. Get employees
curl.exe http://localhost:3000/api/employees

# 5. Check employee availability
curl.exe "http://localhost:3000/api/availability?employee_id=emp-001"

# 6. Book appointment
curl.exe -X POST http://localhost:3000/api/appointments \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -d '{
    "date":"2024-02-20",
    "start_time":"10:00",
    "end_time":"11:00",
    "service_ids":["srv_001"]
  }'

# 7. View appointments
curl.exe http://localhost:3000/api/appointments \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"

# 8. Logout current refresh token
curl.exe -X POST http://localhost:3000/api/revoke \
  -H "Authorization: Bearer YOUR_REFRESH_TOKEN"

# 9. Logout all sessions/devices
curl.exe -X POST http://localhost:3000/api/logout-all \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

**Note**: On Linux/macOS, use `curl` instead of `curl.exe`

---