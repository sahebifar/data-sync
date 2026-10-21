# ---------- build stage ----------
FROM golang:1.27.1-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /out/data-sync .

# ---------- runtime stage ----------
FROM alpine:3.20

RUN adduser -D -u 10001 app
USER app
WORKDIR /app

COPY --from=builder /out/data-sync .

EXPOSE 8080
ENTRYPOINT ["/app/data-sync"]