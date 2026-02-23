# Quickstart: Hello World Website

**Feature**: 001-hello-world-website

## Prerequisites

- Go 1.24+
- Docker

## Build & Run Locally

```bash
cd implementations
go build -o server .
PORT=8000 ./server
```

The server starts on the configured port. Visit
`http://localhost:8000` to see the Hello World page.

## Build & Run with Docker

```bash
cd implementations
docker build -t tabular-api .
docker run -p 8080:8000 -e PORT=8000 tabular-api
```

Visit `http://localhost:8080`.

## Run Integration Tests

```bash
./bin/test.sh
```

This builds the Docker image, starts a container, and
verifies the output contains "Hello, World!".

## Configuration

| Variable | Required | Description |
|----------|----------|-------------|
| `PORT` | Yes | TCP port the server listens on |

If `PORT` is not set, the server logs an error and exits
with code 1.

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/` | Hello World HTML page |
| * | Any other | 404, empty body |
