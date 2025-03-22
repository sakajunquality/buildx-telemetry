FROM golang:1.24.1 as builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN go build -o app cmd/main.go

FROM gcr.io/distroless/base-debian12:nonroot
COPY --from=builder /app/app /bin/app
CMD ["/bin/app"]
