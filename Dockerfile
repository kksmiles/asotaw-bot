# Stage 1: Build the Go application
FROM golang:latest AS builder

WORKDIR /app

# Copy your Go source code to the container
COPY . .

# Install FFmpeg and Opus libraries
RUN apt-get update && \
    apt-get install -y ffmpeg && \
    apt-get install -y libopus-dev && \
    apt-get install -y pkg-config && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Build the Go application
RUN go build -o myapp

# Stage 2: Create the final image
FROM ubuntu:20.04

# Copy the built Go application from the previous stage
COPY --from=builder /app/myapp /usr/local/bin/myapp

# Install youtube-dl
RUN apt-get update && \
    apt-get install -y wget && \
    wget https://yt-dl.org/downloads/latest/youtube-dl -O /usr/local/bin/youtube-dl && \
    chmod +x /usr/local/bin/youtube-dl && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Set the working directory
WORKDIR /app

# Define any environment variables if needed
# ENV MY_ENV_VAR=value

# Expose any necessary ports
# EXPOSE 8080

# Run your Go application
CMD ["myapp"]
