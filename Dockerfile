# ---- build stage ----
FROM golang:1.23 AS build
WORKDIR /app

# Cache dependencies first
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Copy OpenAPI file for distribution
COPY openapi.yaml /openapi.yaml

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/api ./cmd/api

# ---- run stage ----
FROM gcr.io/distroless/base-debian12

ENV HTTP_ADDR=:8080
EXPOSE 8080

COPY --from=build /bin/api /bin/api
COPY --from=build /openapi.yaml /openapi.yaml

# Run as non-root user
USER nonroot:nonroot

ENTRYPOINT ["/bin/api"]
