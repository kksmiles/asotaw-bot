FROM golang:latest

WORKDIR /app

COPY . .

# Install FFmpeg and Opus libraries
RUN apt-get update && \
    apt-get install -y ffmpeg && \
    apt-get install -y libopus-dev && \
    apt-get install -y pkg-config && \
    apt-get install -y wget && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Install yt-dlp
RUN wget https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp -O /usr/local/bin/yt-dlp
RUN chmod a+rx /usr/local/bin/yt-dlp

# Build
RUN go build main.go

# Run
CMD ["./main"]
