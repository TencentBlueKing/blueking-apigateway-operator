FROM golang:1.20 AS builder

COPY ./ /app
WORKDIR /app

ARG BINARY=micro-gateway-operator

RUN make build && chmod +x ./build/${BINARY}

FROM debian:bullseye-slim

ARG BINARY=micro-gateway-operator

RUN mkdir -p /app/logs
COPY --from=builder /app/build/${BINARY} /app/${BINARY}
RUN chmod 755 /app/${BINARY}

CMD ["/app/micro-gateway-operator", "--config=/app/config.yaml"]