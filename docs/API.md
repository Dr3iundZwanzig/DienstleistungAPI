# DienstleistungAPI Documentation

## Overview

DienstleistungAPI is a RESTful API for managing appointments, employees, availability, and services. It uses JWT-based authentication, refresh-token rotation, and role-based access checks for staff/admin endpoints.

## Base URL

```
http://localhost:{PORT}/api
```

Replace `{PORT}` with the configured port from the `PORT` environment variable.

## Authentication

The API uses JWT access tokens for authenticated requests. Refresh tokens are issued as an `HttpOnly` cookie and can also be supplied via the `Authorization` header.

### Token Types

- Access token: short-lived JWT used for API requests
- Refresh token: longer-lived token used to mint new access tokens

### Bearer Token Format

Include the access token in the `Authorization` header:

```
Authorization: Bearer {token}
```

## Rate Limiting

The API applies per-IP rate limits for authentication flows:

- Login: 10 requests/minute
- Failed login attempts: 5 failed attempts/minute
- Refresh: 30 requests/minute

Violations return `429 Too Many Requests`.

## API Endpoints

### Authentication

#### Login

Create a session and obtain an access token.

```
POST /api/login
```

Request body:
```json
{
  "email": "user@example.com",
  "password": "password123"
}
```

Response `200 OK`:
```json
{
  "id": "uuid",
  "email": "user@example.com",
  "role": "user",
  "session_version": 1,
  "created_at": "2024-01-15T10:30:00Z",
  "token": "eyJhbGciOiJIUzI1NiIs..."
}
```

Notes:
- The API also sets an `HttpOnly` `refresh_token` cookie.
- The refresh token is not returned in the JSON body.

Errors:
- `401 Unauthorized`: invalid credentials
- `429 Too Many Requests`: rate limit exceeded

---

#### Refresh Token

Obtain a new access token using a valid refresh token.

```
POST /api/refresh
Authorization: Bearer {refresh_token}
```

Response `200 OK`:
```json
{
  "token": "new_eyJhbGciOiJIUzI1NiIs..."
}
```

Notes:
- The refresh token can also be supplied via the `refresh_token` cookie.
- The previous refresh token is invalidated after a successful refresh.

Errors:
- `400 Bad Request`: no token found
- `401 Unauthorized`: invalid or expired refresh token
- `429 Too Many Requests`: rate limit exceeded

---

#### Revoke Session

Revoke one refresh token.

```
POST /api/revoke
Authorization: Bearer {refresh_token}
```

Response `204 No Content`.

Errors:
- `400 Bad Request`: no token found
- `500 Internal Server Error`: revocation failed

---

#### Logout All Sessions

Invalidate all active sessions for the authenticated user.

```
POST /api/logout-all
Authorization: Bearer {access_token}
```

Response `204 No Content`.

Errors:
- `401 Unauthorized`: missing or invalid token
- `500 Internal Server Error`: invalidation failed

---

### Users

#### Create User

Register a new user account.

```
POST /api/users
```

Request body:
```json
{
  "email": "newuser@example.com",
  "password": "securepassword"
}
```

Response `201 Created`:
```json
{
  "id": "uuid",
  "email": "newuser@example.com",
  "role": "user",
  "session_version": 1,
  "created_at": "2024-01-15T10:30:00Z"
}
```

Errors:
- `400 Bad Request`: email or password missing
- `409 Conflict`: email already exists
- `500 Internal Server Error`: database or hashing error

---

### Appointments

#### List Appointments

Retrieve all appointments for the authenticated user.

```
GET /api/appointments
Authorization: Bearer {access_token}
```

Response `200 OK`:
```json
{
  "data": [
    {
      "id": "appointment-uuid",
      "created_at": "2024-01-15T10:30:00Z",
      "updated_at": "2024-01-15T10:30:00Z",
      "date": "2024-02-20",
      "start_time": "10:00",
      "end_time": "11:00",
      "employee_name": "Max Mustermann",
      "employee_id": "emp-001",
      "user_id": "user-uuid",
      "services": ["Haarschnitt", "Farbe"],
      "total_duration_minutes": 90,
      "total_price": 79.8
    }
  ]
}
```

Errors:
- `401 Unauthorized`: missing or invalid token

---

#### Create Appointment

Book a new appointment.

```
POST /api/appointments
Authorization: Bearer {access_token}
```

Request body:
```json
{
  "date": "2024-02-20",
  "start_time": "10:00",
  "end_time": "11:00",
  "service_ids": ["srv_001", "srv_002"],
  "employee_id": "emp-001",
  "no_preference": false
}
```

Fields:
- `date` (required): `YYYY-MM-DD`
- `start_time` (required): `HH:MM`
- `end_time` (required): `HH:MM`
- `service_ids` (required): one or more leaf-service IDs
- `employee_id` (optional): specific employee ID
- `no_preference` (optional): if `true`, an employee is resolved automatically

Response `201 Created`:
```json
{
  "id": "appointment-uuid",
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z",
  "date": "2024-02-20",
  "start_time": "10:00",
  "end_time": "11:00",
  "employee_name": "Max Mustermann",
  "employee_id": "emp-001",
  "user_id": "user-uuid",
  "services": ["Haarschnitt", "Farbe"],
  "total_duration_minutes": 90,
  "total_price": 79.8
}
```

Errors:
- `400 Bad Request`: invalid body or missing required fields
- `401 Unauthorized`: missing or invalid token
- `409 Conflict`: selected slot is no longer available

---

#### Update Appointment

Modify an existing appointment.

```
PUT /api/appointments/{id}
Authorization: Bearer {access_token}
```

Request body:
```json
{
  "date": "2024-02-21",
  "start_time": "14:00",
  "end_time": "15:00",
  "employee_id": "emp-001"
}
```

Response `200 OK` with the updated appointment object.

Errors:
- `400 Bad Request`: missing required fields
- `401 Unauthorized`: missing or invalid token
- `404 Not Found`: appointment not found
- `409 Conflict`: new slot is unavailable

---

#### Cancel Appointment

Cancel a single appointment.

```
DELETE /api/appointments/{id}
Authorization: Bearer {access_token}
```

Response `200 OK`:
```json
{
  "message": "Appointment cancelled",
  "id": "appointment-uuid"
}
```

Errors:
- `401 Unauthorized`: missing or invalid token
- `404 Not Found`: appointment not found

---

#### Cancel All Appointments

Cancel all appointments for the authenticated user. This is a test endpoint.

```
DELETE /api/appointments/delete
Authorization: Bearer {access_token}
```

Response `200 OK`:
```json
{
  "message": "Appointment cancelled"
}
```

Notes:
- Intended for testing and cleanup flows.
- Reopens availability slots for the affected appointments.

Errors:
- `401 Unauthorized`: missing or invalid token

---

### Availability

#### Get Employee Availability

Retrieve availability for a specific employee.

```
GET /api/availability?employee_id={employee_id}
```

Query parameter:
- `employee_id` (required): employee ID

Response `200 OK`:
```json
{
  "employee_id": "emp-001",
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z",
  "dates": [
    {
      "date": "2024-02-20",
      "slots": [
        {
          "start_time": "09:00",
          "end_time": "10:00",
          "is_available": true
        }
      ]
    }
  ]
}
```

Errors:
- `400 Bad Request`: missing `employee_id`
- `500 Internal Server Error`: no availability exists or database error

---

#### Set Employee Availability

Create or update the availability schedule for an employee.

```
POST /api/availability
Authorization: Bearer {access_token}
```

Authentication: requires staff or admin role.

Request body:
```json
{
  "employee_id": "emp-001",
  "dates": [
    {
      "date": "2024-02-20",
      "slots": [
        {
          "start_time": "09:00",
          "end_time": "10:00",
          "is_available": true
        }
      ]
    }
  ]
}
```

Response `201 Created`:
```json
{
  "message": "Availability saved",
  "availability": {
    "employee_id": "emp-001",
    "created_at": "2024-01-15T10:30:00Z",
    "updated_at": "2024-01-15T10:30:00Z",
    "dates": []
  }
}
```

Errors:
- `400 Bad Request`: invalid request body or missing values
- `401 Unauthorized`: missing or invalid token
- `403 Forbidden`: staff/admin role required

---

### Employees

#### List Employees

Retrieve all employees.

```
GET /api/employees
```

Response `200 OK`:
```json
{
  "data": [
    {
      "id": "emp-001",
      "name": "Max Mustermann",
      "title": "Senior Stylist",
      "specialties": ["srv_001", "srv_002"],
      "is_active": true
    }
  ]
}
```

---

#### Resolve Employee

Resolve an employee ID from a list of services.

```
POST /api/employees/resolve
Authorization: Bearer {access_token}
```

Request body:
```json
{
  "services": ["Haarschnitt", "Farbe"]
}
```

Response `200 OK`:
```json
{
  "employee_id": "emp-001"
}
```

Errors:
- `400 Bad Request`: invalid request body
- `401 Unauthorized`: missing or invalid token
- `500 Internal Server Error`: resolution failed

---

### Services

#### Get Services Tree

Retrieve the service hierarchy.

```
GET /api/services/tree
```

Response `200 OK`:
```json
{
  "data": [
    {
      "id": "cat_01",
      "name": "Friseur",
      "is_active": true,
      "children": [
        {
          "id": "srv_001",
          "name": "Haarschnitt",
          "description": "Waschen, Schneiden, Föhnen",
          "duration_minutes": 45,
          "price": 39.9,
          "currency": "EUR",
          "is_active": true,
          "children": []
        }
      ]
    }
  ]
}
```

---

### Testing (Development Only)

#### Reset and Seed Database

Reset the database and repopulate it with default seed data.

```
POST /api/test/reset-and-seed
Authorization: Bearer {access_token}
```

Authentication: requires staff or admin role.

Response `201 Created`:
```json
{
  "message": "Database reset and test data seeded",
  "seeded_employees": 3,
  "seeded_services": 8
}
```

Notes:
- Only available when `PLATFORM` is `dev` or `test`.
- This endpoint deletes and recreates the seeded data.

#### Reset and reseed from the terminal

If you prefer to trigger the same flow directly from the shell, use the API endpoint with `curl` after the server is running:

```bash
curl.exe -X POST http://localhost:{PORT}/api/test/reset-and-seed \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

For a local development setup, make sure the server is started with `PLATFORM=dev` or `PLATFORM=test` so the endpoint is allowed.

Errors:
- `403 Forbidden`: not available in the current platform
- `401 Unauthorized`: missing token or insufficient permissions
- `500 Internal Server Error`: reset or seeding failed

---

## Error Responses

All error responses follow this format:

```json
{
  "error": "Error message describing what went wrong"
}
```

## Common Status Codes

- `200 OK`
- `201 Created`
- `204 No Content`
- `400 Bad Request`
- `401 Unauthorized`
- `403 Forbidden`
- `404 Not Found`
- `409 Conflict`
- `429 Too Many Requests`
- `500 Internal Server Error`

## Example Flow

```bash
# 1. Register user
curl.exe -X POST http://localhost:3000/api/users \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"secure123"}'

# 2. Login
curl.exe -X POST http://localhost:3000/api/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"secure123"}'

# 3. Get services
curl.exe http://localhost:3000/api/services/tree

# 4. Get employees
curl.exe http://localhost:3000/api/employees

# 5. Check availability
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
```
