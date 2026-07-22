# DienstleistungAPI

REST API for appointment booking and service management (users, employees, availability, services, and authentication).

## Tech Stack

- Go 1.25
- SQLite (default database)
- JWT authentication (access + refresh token flow)
- Static frontend served from `/app/`

## Prerequisites

Install the following tools before running the project:

- Go 1.25+
- C compiler toolchain (required by `github.com/mattn/go-sqlite3` CGO build)
	- Windows: MSYS2/MinGW-w64 or TDM-GCC
	- Linux: `build-essential`
	- macOS: Xcode Command Line Tools

## Project Dependencies

Go modules used by this project (from `go.mod`):

- `github.com/alexedwards/argon2id`
- `github.com/golang-jwt/jwt/v5`
- `github.com/google/uuid`
- `github.com/joho/godotenv`
- `github.com/lib/pq`
- `github.com/mattn/go-sqlite3`

## Installation

### Install from source

Clone the repository and enter the project directory:

```bash
git clone https://github.com/Dr3iundZwanzig/DienstleistungAPI.git
cd DienstleistungAPI
```

Download all Go dependencies:

```bash
go mod download
```

Build the project to verify the toolchain and dependencies are working:

```bash
go build .
```

Run the API directly from source:

```bash
go run .
```

### Optional: Install as a local binary

If you already cloned the repository, you can install the binary from the project root with:

```bash
go install .
```

If you want to install it directly from the module path without cloning first, use:

```bash
go install github.com/Dr3iundZwanzig/DienstleistungAPI@latest
```

This installs the executable into your Go bin directory.

Typical locations:

- Windows: `%USERPROFILE%\\go\\bin`
- Linux/macOS: `$HOME/go/bin`

If that directory is in your `PATH`, you can run the installed binary directly.

Notes:

- `go install .` is useful if you want a reusable command in your PATH.
- The module-path form is useful when installing directly from the repository without cloning first.
- `go install github.com/Dr3iundZwanzig/DienstleistungAPI@latest` installs only the compiled binary, not the repository contents.
- If you use the module-path install, you must still provide the runtime files yourself, especially `.env` and the frontend files referenced by `FILEPATH_ROOT`.
- For local development, `go run .` is usually the simplest workflow.
- The app still requires your `.env` values and a valid `FILEPATH_ROOT` path when you run the installed binary.

### Windows PowerShell quick setup

Add the default Go bin directory to your user `PATH`:

```powershell
$goBin = Join-Path $env:USERPROFILE 'go\bin'
$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')

if (($userPath -split ';') -notcontains $goBin) {
	[Environment]::SetEnvironmentVariable(
		'Path',
		($userPath.TrimEnd(';') + ';' + $goBin).TrimStart(';'),
		'User'
	)
}
```

Open a new PowerShell window after updating `PATH`.

Create a `.env` file in the project root:

```powershell
@'
DB_PATH=./database/dienstleistung.db
JWT_SECRET=replace-with-a-secure-secret
PLATFORM=dev
FILEPATH_ROOT=./app
PORT=8091

JWT_ISSUER=dienstleistung-api
JWT_AUDIENCE=dienstleistung-api-users

ACCESS_TOKEN_TTL=24h
REFRESH_TOKEN_TTL=720h
REFRESH_ACCESS_TOKEN_TTL=15m

LOGIN_RATE_LIMIT_PER_MINUTE=10
LOGIN_FAILED_RATE_LIMIT_PER_MINUTE=5
REFRESH_RATE_LIMIT_PER_MINUTE=30
'@ | Set-Content .env
```

Run the API from the project directory:

```powershell
go run .
```

## Configuration

Create a `.env` file in the project root.

### Required Environment Variables

- `DB_PATH`: SQLite database file path (example: `./database/dienstleistung.db`)
- `JWT_SECRET`: Secret key used to sign and validate JWTs
- `PLATFORM`: Runtime mode (commonly `dev`, `test`, or `prod`)
- `FILEPATH_ROOT`: Path to static frontend files (example: `./app`)
- `PORT`: HTTP port (example: `8091`)

### Optional JWT Claim Variables

- `JWT_ISSUER` (default: `dienstleistung-api`)
- `JWT_AUDIENCE` (default: `dienstleistung-api-users`)

### Optional Auth TTL Variables (Go duration format)

- `ACCESS_TOKEN_TTL` (default: `24h`)
- `REFRESH_TOKEN_TTL` (default: `168h`)
- `REFRESH_ACCESS_TOKEN_TTL` (default: `1h`)

Notes:

- Go duration format is required (`15m`, `1h`, `24h`, `720h`).
- `30d` is not valid in Go duration format; use `720h` instead.

### Optional Rate Limit Variables (requests per minute, per IP)

- `LOGIN_RATE_LIMIT_PER_MINUTE` (default: `10`)
- `LOGIN_FAILED_RATE_LIMIT_PER_MINUTE` (default: `5`)
- `REFRESH_RATE_LIMIT_PER_MINUTE` (default: `30`)

### Example `.env`

```env
DB_PATH=./database/dienstleistung.db
JWT_SECRET=replace-with-a-secure-secret
PLATFORM=dev
FILEPATH_ROOT=./app
PORT=8091

JWT_ISSUER=dienstleistung-api
JWT_AUDIENCE=dienstleistung-api-users

ACCESS_TOKEN_TTL=24h
REFRESH_TOKEN_TTL=720h
REFRESH_ACCESS_TOKEN_TTL=15m

LOGIN_RATE_LIMIT_PER_MINUTE=10
LOGIN_FAILED_RATE_LIMIT_PER_MINUTE=5
REFRESH_RATE_LIMIT_PER_MINUTE=30
```

## Run the API

```bash
go run .
```

When the server starts, the frontend is served at:

- `http://localhost:{PORT}/app/`

The API base path is:

- `http://localhost:{PORT}/api`

## API Documentation

For endpoint details, request/response schemas, and examples, see:

- `docs/API.md`