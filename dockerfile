# Stage 1: The builder stage
# We use a specific version of the Go image to ensure a consistent build environment.
FROM golang:1.22.4-alpine AS builder

# Set the current working directory inside the container.
# This is where all the subsequent commands will be executed.
WORKDIR /app

# Copy the Go module files to the working directory.
# This allows Docker to cache the module download step,
# which speeds up subsequent builds if the dependencies haven't changed.
COPY go.mod go.sum ./

# Download all the required Go modules.
# We use `go mod download` to fetch the dependencies defined in go.mod.
RUN go mod download

# Copy the rest of the application source code into the container.
COPY . .

# Build the Go application binary.
# The `-o` flag specifies the output file name, `weather`.
# The `-ldflags -s -w` flags reduce the size of the final binary by removing
# the debug information.
# The `./cmd/cli` is the entrypoint to the application.
# Update the path to your main package if it is different.
RUN go build -o weather .

# Stage 2: The final, minimal image
# We use a lightweight base image to reduce the final image size.
# Alpine Linux is a popular choice for this.
FROM alpine:latest as Runner

# Set the current working directory to the root for simplicity.
WORKDIR /

# Copy the built binary from the builder stage to the final image.
# We also copy the certificates to ensure HTTPS connections work.
COPY --from=builder /app/weather /usr/local/bin/weather
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Define the command to run when the container starts.
# This executes the compiled Go binary.
ENTRYPOINT ["weather"]
