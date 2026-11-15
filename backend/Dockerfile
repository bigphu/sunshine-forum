# syntax=docker/dockerfile:1

ARG GO_VERSION=1.26.4
ARG AIR_VERSION=v1.67.4

# --- Base: toolchain + module cache (dev và build dùng chung) ---
FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine AS base
WORKDIR /src
ENV GOTOOLCHAIN=local
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# --- Dev container (live reload) ---
FROM base AS dev
ARG AIR_VERSION
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go install github.com/air-verse/air@${AIR_VERSION}
EXPOSE 8080
CMD ["air", "-c", "air.toml"]

# --- Build: server, worker, migrate, seed vào /out ---
FROM base AS build
ARG TARGETOS
ARG TARGETARCH
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=bind,target=. \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" -o /out/ ./cmd/server ./cmd/worker ./cmd/migrate ./cmd/seed

# --- Production: một image, chọn binary bằng entrypoint (/app/worker, /app/migrate) ---
FROM gcr.io/distroless/static-debian12:nonroot AS prod
COPY --from=build /out/ /app/
EXPOSE 8080
ENTRYPOINT ["/app/server"]
