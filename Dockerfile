# ---- build stage ----
FROM golang:1.23-alpine AS build
WORKDIR /src

# Cache module downloads.
COPY go.mod go.sum ./
RUN go mod download

# Build a static binary (modernc.org/sqlite is pure Go, so no CGO needed).
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "-s -w" -o /out/vmt .

# ---- runtime stage ----
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata wget && \
    addgroup -S vmt && adduser -S -G vmt vmt

COPY --from=build /out/vmt /usr/local/bin/vmt

ENV VMT_DATA_DIR=/data \
    VMT_ADDR=:8080
# NOTE: intentionally NO `VOLUME ["/data"]`. A declared VOLUME makes Docker
# create a fresh ANONYMOUS volume at /data whenever the container is recreated
# without an explicit mount — which silently strands data. Always mount the
# named `vmt_data` volume via compose (see docker-compose*.yml) instead.
RUN mkdir -p /data && chown -R vmt:vmt /data
USER vmt

EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8080/login >/dev/null 2>&1 || exit 1

ENTRYPOINT ["/usr/local/bin/vmt"]
