FROM golang as builder

WORKDIR /rnelson3-server
COPY . .

RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app .

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

WORKDIR /rnelson3-server
COPY --from=builder /rnelson3-server/app .

CMD ["./app"]
EXPOSE 8080