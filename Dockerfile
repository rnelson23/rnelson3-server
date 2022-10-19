FROM golang:latest

WORKDIR reddit-server

COPY . .

CMD ["go", "run", "main.go"]