# --- Development: hot reload with Air (source is bind-mounted at /app) ---
FROM golang:1.24-alpine AS dev
WORKDIR /app
RUN go install github.com/air-verse/air@v1.61.7
COPY go.mod ./
RUN go mod download
EXPOSE 8000
CMD ["air", "-c", ".air.toml"]

# --- Production build ---
FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY *.go ./
RUN CGO_ENABLED=0 go build -o /server .

FROM alpine:3.21 AS prod
COPY --from=build /server /server
EXPOSE 8000
CMD ["/server"]
