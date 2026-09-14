# syntax=docker/dockerfile:1
FROM node:22-bookworm AS web
WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.25-bookworm AS build
WORKDIR /src
ARG ACAHTI_VERSION=0.1.0
COPY go.mod go.sum ./
COPY vendor ./vendor
COPY cmd ./cmd
COPY internal ./internal
COPY skills ./skills
COPY --from=web /web/dist ./internal/web/dist
RUN CGO_ENABLED=0 go build -mod=vendor -trimpath \
  -ldflags="-s -w -X acahti/internal/config.Version=${ACAHTI_VERSION}" \
  -o /acahti-gateway ./cmd/acahti-gateway

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata su-exec
COPY --from=build /acahti-gateway /usr/local/bin/acahti-gateway
COPY scripts/gateway-entrypoint.sh /usr/local/bin/gateway-entrypoint
RUN chmod 755 /usr/local/bin/gateway-entrypoint
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/gateway-entrypoint"]
