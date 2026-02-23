# Build stage
FROM golang:1.24-alpine AS build

WORKDIR /app

COPY go.mod ./
COPY main.go ./

RUN CGO_ENABLED=0 GOOS=linux go build -o server .

# Runtime stage
FROM scratch

COPY --from=build /app/server /server

ENTRYPOINT ["/server"]
