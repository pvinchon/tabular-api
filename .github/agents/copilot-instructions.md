# main Development Guidelines

Auto-generated from all feature plans. Last updated: 2026-02-23

## Active Technologies

- Go 1.24 + stdlib (net/http, log/slog, os) (001-hello-world-website)
- Go 1.24 + golang-jwt/jwt/v4 v4.5.2 + Firebase JS SDK v11.x via CDN (002-firebase-auth)

## Project Structure

implementations/
  Dockerfile
  go.mod
  go.sum
  main.go

## Commands

Build: cd implementations && go build -o server .
Run locally: PORT=8000 FIREBASE_PROJECT_ID=... FIREBASE_API_KEY=... FIREBASE_AUTH_DOMAIN=... ./server
Docker: docker compose up --build
Integration tests: ./bin/test.sh

## Code Style

Go 1.24: Follow standard conventions. Single main.go file.
Inline HTML strings. Structured JSON logging via log/slog.

## Recent Changes

- 002-firebase-auth: Added Firebase Authentication (Google sign-in, profile page, JWT verification)
- 001-hello-world-website: Initial Hello World page with Go stdlib HTTP server

<!-- MANUAL ADDITIONS START -->
<!-- MANUAL ADDITIONS END -->
