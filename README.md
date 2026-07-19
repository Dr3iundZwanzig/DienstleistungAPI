# DienstleistungAPI

## Environment Variables

## Token Overview

- Access token: Short-lived JWT used to authenticate protected API requests.
- Refresh token: Longer-lived token stored in the database and used to request a new access token via /api/refresh.
- Revoked/expired refresh token: Cannot be used to mint new access tokens, so the user must log in again.

Required variables:

- DB_PATH
- JWT_SECRET
- PLATFORM
- FILEPATH_ROOT
- PORT

Optional auth TTL variables (Go duration format):

- ACCESS_TOKEN_TTL (default: 720h): Lifetime of the access token returned after login.
- REFRESH_TOKEN_TTL (default: 1440h): Lifetime of the refresh token used to request new access tokens.
- REFRESH_ACCESS_TOKEN_TTL (default: 1h): Lifetime of the access token returned by the refresh endpoint.

Examples:

- ACCESS_TOKEN_TTL=24h
- REFRESH_TOKEN_TTL=30d is not supported by Go duration parser; use 720h instead
- REFRESH_ACCESS_TOKEN_TTL=15m