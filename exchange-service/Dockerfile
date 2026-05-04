FROM golang:1.22-alpine AS build
WORKDIR /src
COPY exchange-service/go.mod ./
RUN go mod download 
COPY exchange-service/ ./
RUN CGO_ENABLED=0 go build -o /out/exchange-service .

FROM alpine:3.20
RUN adduser -D -u 10001 app
WORKDIR /app
COPY --from=build /out/exchange-service /app/exchange-service
COPY data/stocks.json /app/data/stocks.json
COPY data/exchanges.json /app/data/exchanges.json
USER app
EXPOSE 8082
ENV LISTEN_ADDR=":8082" STOCKS_DATA="/app/data/stocks.json" EXCHANGES_DATA="/app/data/exchanges.json"
ENTRYPOINT ["/app/exchange-service"]
