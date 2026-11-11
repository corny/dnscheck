# Build stage
FROM golang:1.27 AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /dnscheck ./cmd/dnscheck

# directories of the volume, writable by the nonroot user of the runtime image
RUN mkdir -p /data/export /data/geoip

# Runtime stage: no shell, CA certificates for the GeoLite2 download
FROM gcr.io/distroless/static-debian13:nonroot

COPY --from=build /dnscheck /dnscheck
COPY domains.txt /etc/dnscheck/domains.txt
COPY --from=build --chown=nonroot:nonroot /data /data

VOLUME /data
EXPOSE 8000

ENTRYPOINT ["/dnscheck"]
CMD ["check", \
     "--syslog", \
     "--domains=/etc/dnscheck/domains.txt", \
     "--export.output=/data/export", \
     "--geoip.dir=/data/geoip"]
