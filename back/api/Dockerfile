# Use the official Golang 1.23 image for building
FROM golang:1.23-alpine as build

WORKDIR /app

# Copy go module files and download dependencies
COPY go.mod go.sum ./
RUN go mod tidy

# Copy the source code
COPY . .

# Build the binary with optimizations
RUN CGO_ENABLED=0 go build -o api-server .

# Use a minimal image for production (scratch or alpine)
FROM alpine:latest

WORKDIR /root/

# Install necessary libraries
RUN apk add --no-cache ca-certificates

# Copy the compiled binary from the build stage
COPY --from=build /app/api-server .

# Expose the API port
EXPOSE 3000

# Run the API server
CMD ["./api-server"]
