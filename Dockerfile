FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
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
