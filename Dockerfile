# Use a Go version >= 1.24.2
FROM golang:1.24.2

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . .

RUN go build -o main .

EXPOSE 3000

CMD ["./main"]
