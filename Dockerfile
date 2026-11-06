FROM golang:1.27-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" \
    -o /trip-service ./cmd/trip-service

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /trip-service /trip-service

USER nonroot:nonroot
EXPOSE 8080

ENTRYPOINT ["/trip-service"]
