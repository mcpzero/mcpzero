FROM golang:1.22-bookworm AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 ./scripts/build-all.sh

FROM scratch AS export
COPY --from=builder /src/dist /dist
