# Use the official Go image to build the application
FROM golang:1.22.4-alpine AS builder

# Set the working directory
WORKDIR /app

# Copy the source code
COPY . .

# Build the Go application
RUN go mod tidy
RUN CGO_ENABLED=0 go build -o /weather-app .

# Use a minimal image for the final container
FROM alpine:3.18

# Set the working directory
WORKDIR /

# Copy the binary from the builder stage
COPY --from=builder /weather-app /weather-app

# Expose the port the web server will listen on
EXPOSE 8080

# The command to run the application
# It will keep running as a web server
CMD ["/weather-app"]