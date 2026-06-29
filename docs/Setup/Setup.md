# Setup & Installation

## Prerequisites
*   **Go** 1.22+
*   **C Compiler** (`gcc` — required for `mattn/go-sqlite3` CGO bindings)
*   **Git**
*   **Docker**

## Initializing the Project
```bash
git clone <repo-url> && cd forum
go mod init forum
go get github.com/mattn/go-sqlite3 golang.org/x/crypto/bcrypt github.com/google/uuid
```

## Environment Variables
*   `PORT`: Sets the server network port (Default: `8080`)
*   `DB_PATH`: Sets the local path to the SQLite database file (Default: `./forum.db`)

## Core Commands

| Action | Command |
| :--- | :--- |
| **Run Local Server** | `go run ./cmd/server` |
| **Test & Sanitize** | `go fmt ./... && go vet ./... && go test ./... -v -cover -race` |
| **Containerize App** | `docker build -t forum:latest .` |
| **Run Container** | `docker run -p 8080:8080 forum:latest` |
