##########
# Builder #
##########
FROM golang:1.24-alpine AS builder

WORKDIR /src

# Build deps
RUN apk add --no-cache ca-certificates tzdata git

# Cache module deps
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source
COPY . .

# Build static binaries
ENV CGO_ENABLED=0 GOOS=linux
RUN mkdir -p /out/bin /out/tests \
    && go build -ldflags="-s -w" -o /out/bin/server . \
    && go build -ldflags="-s -w" -o /out/bin/testrunner ./cmd/testrunner

# Compile per-package test binaries (only for packages that have *_test.go)
RUN set -eux; \
  for pkg in $(go list ./...); do \
    dir=$(go list -f "{{.Dir}}" "$pkg"); \
    if ls "$dir"/*_test.go >/dev/null 2>&1; then \
      rel=${dir#$(pwd)/}; \
      mkdir -p "/out/tests/$rel"; \
      go test -c -o "/out/tests/$rel.test" "$pkg"; \
    fi; \
  done

################
# Final image  #
################
FROM gcr.io/distroless/base-debian12:nonroot

WORKDIR /app

# Copy minimal runtime and artifacts
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /out/bin/server /app/bin/server
COPY --from=builder /out/bin/testrunner /app/bin/testrunner
COPY --from=builder /out/tests /app/tests

# Expose gRPC and HTTP ports
EXPOSE 50051 8080

# Run as nonroot (provided by distroless)
USER nonroot:nonroot

# Run the server
CMD ["/app/bin/server"]
