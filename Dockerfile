# Start from the latest golang base image.
# We use multi-stage build here to reduce the size of the final image
FROM golang:latest AS builder
# Add Maintainer Info
LABEL maintainer="Stephin Thomas"
# Set the Current Working Directory inside the container
WORKDIR /workload-web-app
# Copy go mod and sum files
COPY go.mod ./
# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download
# Copy the source from the current directory to the Working Directory inside the container
COPY . .
# Build the Go app
RUN make build

FROM alpine:latest
RUN apk --no-cache add ca-certificates
RUN apk --no-cache add curl
WORKDIR /root/
COPY --from=builder /workload-web-app/workload-app .
EXPOSE 8080
CMD ["./workload-app"]