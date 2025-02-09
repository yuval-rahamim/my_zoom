# Use the official Golang image to build the API server
FROM golang:1.20-alpine as build

WORKDIR /app

# Copy the Go module files and download dependencies
COPY go.mod go.sum ./
RUN go mod tidy

# Copy the source code
COPY . .

# Build the API binary
RUN go build -o api-server .

# Use a smaller image for the final container
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
