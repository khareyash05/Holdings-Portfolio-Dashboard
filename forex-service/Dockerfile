FROM golang:1.22-alpine AS build
WORKDIR /src
COPY forex-service/go.mod ./
RUN go mod download || true
COPY forex-service/ ./
RUN CGO_ENABLED=0 go build -o /out/forex-service .

FROM alpine:3.20
RUN adduser -D -u 10001 app
WORKDIR /app
COPY --from=build /out/forex-service /app/forex-service
COPY data/forex.json /app/data/forex.json
USER app
EXPOSE 8081
ENV LISTEN_ADDR=":8081" FOREX_DATA="/app/data/forex.json"
ENTRYPOINT ["/app/forex-service"]
