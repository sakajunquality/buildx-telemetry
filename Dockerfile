FROM golang:1.24.1 as builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/buildx-telemetry ./cmd

FROM gcr.io/distroless/base-debian12:nonroot
COPY --from=builder /app/bin/buildx-telemetry /bin/buildx-telemetry

ENTRYPOINT ["/bin/buildx-telemetry"]
