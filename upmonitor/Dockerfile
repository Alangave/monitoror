# Build stage: UI (Node.js 24)
FROM node:24-alpine AS ui
WORKDIR /src
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# Build stage: static Go binary (latest Go)
FROM golang:1.27-alpine AS build
WORKDIR /src
ENV CGO_ENABLED=0
COPY go.mod go.sum* ./
COPY cmd/ cmd/
COPY internal/ internal/
COPY *.go ./
COPY --from=ui /src/dist ./web/dist
RUN go build -trimpath -ldflags="-s -w" -o /out/upmonitor ./cmd/upmonitor

# Runtime stage: distroless, non-root (rootless)
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/upmonitor /upmonitor
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/upmonitor"]
