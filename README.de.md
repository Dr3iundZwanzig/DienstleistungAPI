# DienstleistungAPI

Sprache: Deutsch | [English](README.md)

REST-API für Terminbuchung und Serviceverwaltung (Benutzer, Mitarbeiter, Verfügbarkeiten, Services und Authentifizierung).

## Technologie-Stack

- Go 1.25
- SQLite (Standarddatenbank)
- JWT-Authentifizierung (Access- + Refresh-Token-Flow)
- Statisches Frontend aus `/app/`

## Voraussetzungen

Installieren Sie die folgenden Werkzeuge, bevor Sie das Projekt starten:

- Go 1.25+
- C-Compiler-Toolchain (erforderlich durch den CGO-Build von `github.com/mattn/go-sqlite3`)
	- Windows: MSYS2/MinGW-w64 oder TDM-GCC
	- Linux: `build-essential`
	- macOS: Xcode Command Line Tools

### Windows-CGO-Setup

Wenn Sie `MSYS2/MinGW-w64` oder `TDM-GCC` installieren, sind meist ein paar zusätzliche Schritte notwendig, bevor Go die SQLite-Abhängigkeit bauen kann.

Nach der Installation des Compilers:

1. Stellen Sie sicher, dass `gcc` installiert ist.
2. Fügen Sie das `bin`-Verzeichnis des Compilers zu Ihrer Windows-`PATH`-Variable hinzu.
3. Öffnen Sie ein neues Terminal.
4. Prüfen Sie, ob `gcc` verfügbar ist.
5. Prüfen Sie, ob das Projekt erfolgreich gebaut wird.

Beispiel für MSYS2 UCRT64:

```powershell
pacman -S --needed mingw-w64-ucrt-x86_64-gcc
```

Typischer MSYS2-Compilerpfad:

- `C:\msys64\ucrt64\bin`

So fügen Sie diesen Ordner zu Ihrem Windows-`PATH` hinzu:

1. Drücken Sie die Windows-Taste und suchen Sie nach `Umgebungsvariablen`.
2. Öffnen Sie `Systemumgebungsvariablen bearbeiten`.
3. Klicken Sie auf `Umgebungsvariablen...`.
4. Unter `Benutzervariablen` wählen Sie `Path` und klicken auf `Bearbeiten`.
5. Klicken Sie auf `Neu` und fügen Sie den Compiler-`bin`-Pfad hinzu, z. B. `C:\msys64\ucrt64\bin`.
6. Bestätigen Sie alle Dialoge mit `OK`.
7. Öffnen Sie ein neues PowerShell-Fenster.

Installation prüfen:

```powershell
gcc --version
go version
go build .
```

Wenn Sie stattdessen TDM-GCC nutzen, ist der Ablauf gleich: `bin`-Verzeichnis zu `PATH` hinzufügen, neues Terminal öffnen und `gcc --version` prüfen, bevor Sie `go build .` ausführen.

Typischer TDM-GCC-Compilerpfad:

- `C:\TDM-GCC-64\bin`

## Go installieren

Wenn Go noch nicht installiert ist, nutzen Sie eine der folgenden Optionen.

### Windows

Laden Sie den Installer von der offiziellen Go-Webseite herunter und führen Sie ihn aus:

- https://go.dev/dl/

Oder installieren Sie mit `winget`:

```powershell
winget install GoLang.Go
```

Öffnen Sie nach der Installation ein neues Terminal und prüfen Sie:

```powershell
go version
```

### Linux

Installieren Sie Go über den Paketmanager Ihrer Distribution oder über das offizielle Tar-Archiv.

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

Installation prüfen:

```bash
go version
```

### macOS

Installieren Sie mit Homebrew:

```bash
brew install go
```

Oder laden Sie den offiziellen Installer herunter:

- https://go.dev/dl/

Installation prüfen:

```bash
go version
```

### Empfohlene Versionsprüfung

Dieses Projekt zielt auf Go 1.25 ab. Prüfen Sie Ihre Version vor dem Fortfahren:

```bash
go version
```

## Projektabhängigkeiten

Von diesem Projekt genutzte Go-Module (aus `go.mod`):

- `github.com/alexedwards/argon2id`
- `github.com/golang-jwt/jwt/v5`
- `github.com/google/uuid`
- `github.com/joho/godotenv`
- `github.com/lib/pq`
- `github.com/mattn/go-sqlite3`

## Installation

### Git installieren

Git muss installiert sein, bevor Sie dieses Repository klonen können.

Prüfen Sie, ob Git bereits installiert ist:

```bash
git --version
```

Wenn der Befehl nicht gefunden wird, installieren Sie Git für Ihre Plattform:

#### Windows

Installieren Sie über die offizielle Webseite:

- https://git-scm.com/download/win

Oder installieren Sie mit `winget`:

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

Installieren Sie mit Homebrew:

```bash
brew install git
```

Oder installieren Sie die Xcode Command Line Tools:

```bash
xcode-select --install
```

Öffnen Sie nach der Installation ein neues Terminal und prüfen Sie:

```bash
git --version
```

### Aus dem Quellcode installieren

Repository klonen und in das Projektverzeichnis wechseln:

```bash
git clone https://github.com/Dr3iundZwanzig/DienstleistungAPI.git
cd DienstleistungAPI
```

Alle Go-Abhängigkeiten herunterladen:

```bash
go mod download
```

Projekt bauen, um Toolchain und Abhängigkeiten zu prüfen:

```bash
go build .
```

API direkt aus dem Ordner starten:

```bash
go run .
```

### Optional: Als lokale Binärdatei installieren

Wenn Sie das Repository bereits geklont haben, können Sie die Binärdatei im Projekt-Root installieren mit:

```bash
go install .
```

Wenn Sie direkt über den Modulpfad installieren möchten, ohne vorher zu klonen:

```bash
go install github.com/Dr3iundZwanzig/DienstleistungAPI@latest
```

Das installiert die ausführbare Binärdatei in Ihr Go-`bin`-Verzeichnis.

Typische Orte:

- Windows: `%USERPROFILE%\\go\\bin`
- Linux/macOS: `$HOME/go/bin`

Wenn dieses Verzeichnis in Ihrem `PATH` ist, können Sie die installierte Binärdatei direkt ausführen.

Hinweise:

- `go install .` ist nützlich, wenn Sie einen wiederverwendbaren Befehl in Ihrem `PATH` möchten.
- Die Modulpfad-Variante ist nützlich, wenn Sie direkt aus dem Repository installieren, ohne zu klonen.
- `go install github.com/Dr3iundZwanzig/DienstleistungAPI@latest` installiert nur die kompilierte Binärdatei, nicht den Repository-Inhalt.
- Wenn Sie über den Modulpfad installieren, müssen Sie die Laufzeitdateien trotzdem selbst bereitstellen, besonders `.env` und die Frontend-Dateien, auf die `FILEPATH_ROOT` zeigt.
- Für lokale Entwicklung ist `go run .` meist der einfachste Workflow.
- Die App benötigt weiterhin gültige `.env`-Werte und einen korrekten `FILEPATH_ROOT`, auch wenn Sie die installierte Binärdatei starten.

## Konfiguration

Erstellen Sie eine `.env`-Datei im Projekt-Root.

### Erforderliche Umgebungsvariablen

- `DB_PATH`: Pfad zur SQLite-Datenbankdatei (Beispiel: `./database/dienstleistung.db`)
- `JWT_SECRET`: Geheimer Schlüssel zum Signieren und Validieren von JWTs
- `PLATFORM`: Runtime-Modus (häufig `dev`, `test` oder `prod`)
- `FILEPATH_ROOT`: Pfad zu statischen Frontend-Dateien (Beispiel: `./app`)
- `PORT`: HTTP-Port (Beispiel: `8091`)

### Optionale JWT-Claim-Variablen

- `JWT_ISSUER` (Standard: `dienstleistung-api`)
- `JWT_AUDIENCE` (Standard: `dienstleistung-api-users`)

### Optionale Auth-TTL-Variablen (Go-Duration-Format)

- `ACCESS_TOKEN_TTL` (Standard: `24h`)
- `REFRESH_TOKEN_TTL` (Standard: `168h`)
- `REFRESH_ACCESS_TOKEN_TTL` (Standard: `1h`)

Hinweise:

- Go-Duration-Format ist erforderlich (`15m`, `1h`, `24h`, `720h`).
- `30d` ist im Go-Duration-Format ungültig; verwenden Sie stattdessen `720h`.

### Optionale Rate-Limit-Variablen (Requests pro Minute, pro IP)

- `LOGIN_RATE_LIMIT_PER_MINUTE` (Standard: `10`)
- `LOGIN_FAILED_RATE_LIMIT_PER_MINUTE` (Standard: `5`)
- `REFRESH_RATE_LIMIT_PER_MINUTE` (Standard: `30`)

### Beispiel `.env`

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

## API starten

```bash
go run .
```

Wenn der Server gestartet ist, wird das Frontend ausgeliefert unter:

- `http://localhost:{PORT}/app/`

Der API-Basispfad ist:

- `http://localhost:{PORT}/api`

## API-Dokumentation

Details zu Endpunkten, Anfrage-/Antwort-Schemas und Beispielen:

- `docs/API.md`