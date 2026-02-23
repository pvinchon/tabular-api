# HTTP API Contract: Hello World Website

**Feature**: 001-hello-world-website
**Date**: 2026-02-23
**Protocol**: HTTP/1.1

## Endpoints

### `GET /`

Serves the Hello World HTML page.

**Request**: No parameters, no headers required.

**Response (200)**:

```
HTTP/1.1 200 OK
Content-Type: text/html; charset=utf-8

<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Hello, World!</title>
</head>
<body>
    <h1>Hello, World!</h1>
</body>
</html>
```

**Invariants**:
- Response body MUST contain the text `Hello, World!`
- Response MUST be valid HTML5
- Response MUST declare `charset=utf-8`
- Response MUST include viewport meta tag

---

### Any undefined route

**Response (404)**:

```
HTTP/1.1 404 Not Found
Content-Length: 0
```

**Invariants**:
- Response body MUST be empty
- Status code MUST be 404

## Error Envelope

This feature serves static content and health checks only.
API-First error envelope (Constitution VIII) will apply to
future data endpoints. For this feature:

- `GET /` returns HTML, not JSON
- Undefined routes return empty 404
- Health and readiness endpoints deferred to a future feature

## Structured Logging

Every request produces a structured JSON log entry:

```json
{
  "time": "2026-02-23T12:00:00.000Z",
  "level": "INFO",
  "msg": "request",
  "request_id": "abc-123",
  "method": "GET",
  "path": "/",
  "status": 200,
  "latency_ms": 0.5
}
```

Fields:
- `time`: ISO 8601 timestamp
- `level`: log level (INFO for requests, ERROR for failures)
- `msg`: log message
- `request_id`: unique identifier per request (UUID or similar)
- `method`: HTTP method
- `path`: request path
- `status`: HTTP response status code
- `latency_ms`: request duration in milliseconds
