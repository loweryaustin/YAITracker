FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata curl

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

RUN curl -sL https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-linux-x64 -o /usr/local/bin/tailwindcss \
    && chmod +x /usr/local/bin/tailwindcss

COPY . .
RUN tailwindcss -i static/css/input.css -o static/css/app.css --minify
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /yaitracker ./cmd/yaitracker

RUN adduser -D -u 1000 yaitracker

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group
COPY --from=builder /yaitracker /yaitracker

USER 1000:1000

EXPOSE 8080

VOLUME ["/data"]

ENTRYPOINT ["/yaitracker"]
CMD ["serve", "--db", "/data/yaitracker.db", "--addr", ":8080"]
