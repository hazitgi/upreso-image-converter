FROM golang:1.22.4

# Set working 
WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

# Install necessary build dependencies
RUN apt-get update && \
    apt-get install -y gcc && \
    apt-get install -y musl-dev && \
    rm -rf /var/lib/apt/lists/*

# Enable cgo for Go builds
ENV CGO_ENABLED=1

# Build the Go application
RUN go build -o main .
CMD ["./main"]