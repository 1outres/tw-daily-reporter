FROM golang:1.22.4 AS builder

WORKDIR /app

COPY . .

WORKDIR /app

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o twdr cmd/twdr/main.go

FROM gcr.io/distroless/base-debian11 AS runner

WORKDIR /

COPY --from=builder /app/twdr /twdr

USER nonroot:nonroot

ENTRYPOINT ["/twdr"]
