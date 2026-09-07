# syntax=docker/dockerfile:1.7
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -buildvcs=false \
      -ldflags="-s -w" -o /tka-api ./cmd/api

FROM gcr.io/distroless/static-debian12:nonroot
ENV APP_PORT=8080
COPY --from=build --chown=nonroot:nonroot /tka-api /tka-api
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/tka-api"]
