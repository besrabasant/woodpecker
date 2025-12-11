# Use an official Go image
FROM golang:1.25-alpine AS base

RUN apk add --update \
  bash curl openssh-keygen openssl \
  && rm -rf /var/cache/apk/* \
  ## Install Air for live reloading
  && go install github.com/air-verse/air@v1.61.7

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# Expose application port
EXPOSE 1339

# Set Air as the entrypoint for development
ENTRYPOINT ["/entrypoint.sh"]

CMD ["air", "-c", ".air.toml"]
