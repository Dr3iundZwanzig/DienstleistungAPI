# DienstleistungAPI

Language: English | [Deutsch](README.de.md)

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

### Windows CGO setup

If you install `MSYS2/MinGW-w64` or `TDM-GCC`, there are usually a few extra steps before Go can build the SQLite dependency.

After installing the compiler, do the following:

1. Make sure `gcc` is installed.
2. Add the compiler `bin` directory to your Windows `PATH`.
3. Open a new terminal.
4. Verify that `gcc` is available.
5. Verify that the project builds.

Example for MSYS2 UCRT64:

```powershell
pacman -S --needed mingw-w64-ucrt-x86_64-gcc
```

Typical MSYS2 compiler path:

- `C:\msys64\ucrt64\bin`

How to add this folder to your Windows `PATH`:

1. Press the Windows key and search for `Environment Variables`.
2. Open `Edit the system environment variables`.
3. Click `Environment Variables...`.
4. Under `User variables`, select `Path` and click `Edit`.
5. Click `New` and add the compiler `bin` path, for example `C:\msys64\ucrt64\bin`.
6. Confirm with `OK` on all dialogs.
7. Open a new PowerShell window.

To verify the setup:

```powershell
gcc --version
go version
go build .
```

If you use TDM-GCC instead, the process is the same: add its `bin` directory to `PATH`, open a new terminal, and verify `gcc --version` before running `go build .`.

Typical TDM-GCC compiler path:

- `C:\TDM-GCC-64\bin`


## Install Go

If Go is not installed yet, use one of the following options.

### Windows

Download the installer from the official Go website and run it:

- https://go.dev/dl/

Or install with `winget`:

```powershell
winget install GoLang.Go
```

After installation, open a new terminal and verify:

```powershell
go version
```

### Linux

Install Go from your distribution package manager or from the official tarball.

Ubuntu/Debian:

```bash
sudo apt update
sudo apt install golang-go
```

Fedora:

```bash
sudo dnf install golang
```

Arch Linux:

```bash
sudo pacman -S go
```

Verify the installation:

```bash
go version
```

### macOS

Install with Homebrew:

```bash
brew install go
```

Or download the official installer:

- https://go.dev/dl/

Verify the installation:

```bash
go version
```

### Recommended version check

This project targets Go 1.25. Confirm your version before continuing:

```bash
go version
```

## Project Dependencies

Go modules used by this project (from `go.mod`):

- `github.com/alexedwards/argon2id`
- `github.com/golang-jwt/jwt/v5`
- `github.com/google/uuid`
- `github.com/joho/godotenv`
- `github.com/lib/pq`
- `github.com/mattn/go-sqlite3`

## Installation

### Install Git

Git must be installed before you can clone this repository.

Verify whether Git is already installed:

```bash
git --version
```

If the command is not found, install Git for your platform:

#### Windows

Install from the official website:

- https://git-scm.com/download/win

Or install with `winget`:

```powershell
winget install Git.Git
```

#### Linux

Ubuntu/Debian:

```bash
sudo apt update
sudo apt install git
```

Fedora:

```bash
sudo dnf install git
```

Arch Linux:

```bash
sudo pacman -S git
```

#### macOS

Install with Homebrew:

```bash
brew install git
```

Or install Xcode Command Line Tools:

```bash
xcode-select --install
```

After installation, open a new terminal and verify:

```bash
git --version
```

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