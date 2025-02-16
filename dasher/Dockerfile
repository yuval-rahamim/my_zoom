# Use an official FFmpeg image as a base
FROM jrottenberg/ffmpeg:latest

# Install additional dependencies if needed
RUN apt-get update && apt-get install -y \
    nginx \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Create a directory to store videos and DASH segments
RUN mkdir -p /app/videos /app/dash

# Set the working directory
WORKDIR /app

# Expose ports for streaming (for example, 5000 for HTTP)
EXPOSE 5000

# Copy your server files (we'll configure Nginx or a simple HTTP server to serve the files)
COPY . /app

# Start Nginx or another HTTP server to serve the files
CMD ["nginx", "-g", "daemon off;"]
