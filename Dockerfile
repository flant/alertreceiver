FROM golang:1.25-bookworm AS builder

ARG versionflags

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -v -a -tags netgo -ldflags="-extldflags '-static' -s -w $versionflags" -o alertreceiver cmd/main.go

FROM debian:bookworm-slim

WORKDIR /app

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get upgrade -y && apt-get install -y --no-install-recommends \
        ca-certificates \
        && rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/alertreceiver /app/alertreceiver

RUN chmod +x /app/alertreceiver

ENV PATH="/app:${PATH}"

CMD ["/app/alertreceiver"]

