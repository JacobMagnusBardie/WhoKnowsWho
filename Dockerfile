# Build stage: full Go toolchain, discarded afterwards
  FROM golang:1.26-alpine AS build
  WORKDIR /src/backend
  # Copy go.mod/go.sum first so the dependency layer is cached between builds
  # checks changes, if changes -> re-download all dependencies.
  COPY backend/go.mod backend/go.sum ./
  RUN go mod download
  COPY backend/ ./
  RUN CGO_ENABLED=0 go build -o /whoknows-app .

  # Runtime stage: only the binary and the frontend files
  FROM alpine:3.22
  WORKDIR /app/backend
  COPY --from=build /whoknows-app ./
  COPY frontend/ /app/frontend/
  EXPOSE 8080
  CMD ["./whoknows-app"]