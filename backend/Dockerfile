FROM golang:1.22-alpine AS build
WORKDIR /src
COPY backend/ ./
RUN go mod tidy
RUN CGO_ENABLED=0 go build -o /out/server ./cmd/server

FROM alpine:3.20
RUN adduser -D -u 10001 app && apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /out/server /app/server
COPY data/stocks.json /app/data/stocks.json
COPY data/exchanges.json /app/data/exchanges.json
USER app
EXPOSE 8080
ENV LISTEN_ADDR=":8080"
ENTRYPOINT ["/app/server"]
