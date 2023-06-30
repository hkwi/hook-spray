FROM golang AS builder
ARG http_proxy https_proxy HTTP_PROXY HTTPS_PROXY
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /hook-spray

FROM alpine
COPY --from=builder /hook-spray /hook-spray
ENTRYPOINT ["/hook-spray"]
