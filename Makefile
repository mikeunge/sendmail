SRC = main.go
BINARY = sendmail

build:
	go build -o $(BINARY) $(SRC)

run: build
	./$(BINARY) --help

clean:
	rm -f $(BINARY)

.PHONY: build run clean