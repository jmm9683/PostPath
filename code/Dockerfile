FROM golang:1.23

WORKDIR /app
COPY . .

# Enable CGO
ENV CGO_ENABLED=1
RUN apt update && apt install -y gcc libc6-dev

RUN go build -o main .

CMD ["./main"]
