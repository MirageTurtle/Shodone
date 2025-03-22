.PHONY: build clean run

# Binary name
BINARY_NAME=shodone

# Build the application
build:
	go build -o $(BINARY_NAME) ./cmd/server

# Clean build files
clean:
	go clean
	rm -f $(BINARY_NAME)

# Run the application
run: build
	./$(BINARY_NAME)
