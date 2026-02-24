````markdown
# Quickstart: Firebase Authentication

**Feature**: 002-firebase-auth

## Prerequisites

- Go 1.24+
- Docker
- A Firebase project with Google sign-in enabled
- Firebase project configuration values:
  - Project ID
  - API Key
  - Auth Domain

## Build & Run Locally

```bash
cd implementations
go build -o server .
PORT=8000 \
  FIREBASE_PROJECT_ID=your-project-id \
  FIREBASE_API_KEY=your-api-key \
  FIREBASE_AUTH_DOMAIN=your-project.firebaseapp.com \
  ./server
```

The server starts on the configured port.

- Visit `http://localhost:8000` for the Hello World page.
- Visit `http://localhost:8000/profile` for the profile page
  (sign-in required).

## Build & Run with Docker

```bash
cd implementations
docker build -t tabular-api .
docker run -p 8080:8000 \
  -e PORT=8000 \
  -e FIREBASE_PROJECT_ID=your-project-id \
  -e FIREBASE_API_KEY=your-api-key \
  -e FIREBASE_AUTH_DOMAIN=your-project.firebaseapp.com \
  tabular-api
```

Visit `http://localhost:8080`.

## Run with Docker Compose

```bash
docker compose up --build
```

Requires environment variables set in `compose.yaml` or a
`.env` file.

## Run Integration Tests

```bash
./bin/test.sh
```

## Configuration

| Variable | Required | Description |
|----------|----------|-------------|
| `PORT` | Yes | TCP port the server listens on |
| `FIREBASE_PROJECT_ID` | Yes | Firebase project ID (for server-side token verification) |
| `FIREBASE_API_KEY` | Yes | Firebase API key (injected into client-side JS) |
| `FIREBASE_AUTH_DOMAIN` | Yes | Firebase auth domain (injected into client-side JS) |

All variables are required. If any is missing, the server logs
an error and exits with code 1.

## Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/` | No | Hello World HTML page |
| GET | `/profile` | No (client-enforced) | Profile page HTML shell |
| GET | `/api/me` | Yes (Bearer token) | Authenticated user profile (JSON) |
| * | Any other | No | 404, empty body |

## Authentication Flow

1. Visit `/profile`.
2. Client-side JS checks Firebase auth state.
3. If not signed in, the Google sign-in popup is initiated.
4. After sign-in, JS calls `GET /api/me` with the Firebase ID
   token as a Bearer token.
5. Server verifies the token and returns JSON profile data.
6. JS renders the profile on the page.
7. "Sign out" button ends the session client-side and returns
   to the unauthenticated state.

````
