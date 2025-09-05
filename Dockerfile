# syntax=docker/dockerfile:1.7

# --- Build stage ---
    FROM golang:1.23-alpine AS build
    WORKDIR /src
    RUN apk add --no-cache git ca-certificates
    COPY go.mod go.sum ./
    RUN go mod download
    COPY . .
    # change if main.go moved
    RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /out/api ./cmd/api
    
    # --- Runtime stage (distroless) ---
    FROM gcr.io/distroless/static-debian12
    WORKDIR /app
    COPY --from=build /out/api /app/api
    ENV APP_PORT=8080
    EXPOSE 8080
    USER 65532:65532
    ENTRYPOINT ["/app/api"]
    