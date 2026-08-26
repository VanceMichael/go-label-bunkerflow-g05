FROM golang:1.26-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/bunkerflow ./cmd/server

FROM debian:bookworm-slim
RUN useradd --system --uid 10001 bunkerflow && mkdir -p /var/lib/bunkerflow && chown -R bunkerflow:bunkerflow /var/lib/bunkerflow
WORKDIR /app
COPY --from=build /out/bunkerflow /app/bunkerflow
COPY migrations /app/migrations
USER bunkerflow
ENV DATABASE_URL=/var/lib/bunkerflow/bunkerflow.db HTTP_ADDR=:8080
EXPOSE 8080
ENTRYPOINT ["/app/bunkerflow"]
