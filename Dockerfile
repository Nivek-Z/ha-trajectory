
FROM golang:1.21-alpine AS builder


WORKDIR /app


COPY . .


RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o ha-trajectory ./cmd/server


FROM scratch


COPY --from=builder /app/ha-trajectory /ha-trajectory

EXPOSE 8080

ENTRYPOINT ["/ha-trajectory"]