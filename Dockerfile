# Use an appropriate base image
# FROM golang:1.22.5 as builder
FROM golang:1.22.5

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Copy the source from the current directory to the Working Directory inside the container
COPY . .

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Build the Go app
RUN go build -o main ./cmd/main.go

# Stage 2: Final stage
# FROM alpine:latest

# # Set the working directory for the final image
# WORKDIR /root/

# # Copy the built executable from the builder stage to the final image
# COPY --from=builder /app/main .

# Run the application
CMD ["./main"]