# --- Development: hot reload with Air (source is bind-mounted at /app) ---
FROM golang:1.24-alpine AS dev
WORKDIR /app
RUN go install github.com/air-verse/air@v1.61.7
COPY go.mod go.sum ./
RUN go mod download
EXPOSE 8000
CMD ["air", "-c", ".air.toml"]

