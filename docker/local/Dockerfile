FROM golang:1.20-alpine AS builder

RUN apk add make git

WORKDIR /metachain

ADD go.mod go.sum ./
RUN go mod download

COPY . .

RUN make build

FROM alpine:latest AS runner

RUN apk --no-cache add ca-certificates jq

WORKDIR /metachain

COPY --from=builder /metachain/metachain ./
COPY ./docker/local/metachain.sh ./
COPY ./core-contracts/artifacts ./core-contracts/artifacts

# Expose json-rpc, libp2p and grpc ports
EXPOSE 8545 9632 1478 5001

ENTRYPOINT ["./metachain.sh"]
