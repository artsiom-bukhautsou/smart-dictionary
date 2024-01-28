# Use the official Golang base image
FROM golang:latest

# Set the working directory inside the container
WORKDIR /app

# Copy only the necessary files for dependency download
COPY go.mod go.sum ./

# Download and install any required dependencies
RUN go mod download

# Copy the entire project to the container's workspace
COPY . .

# Build the Go application
RUN go build -o main ./cmd

# Expose the port on which the application will run
EXPOSE 8080

# Command to run the executable
CMD ["./main"]
