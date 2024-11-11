.PHONY: build compress decompress

build:
	@go build -o ./bin/zx0 .

compress:
	@go run main.go -p=0 -f "$(INPUT_FILE)"

decompress:
	@go run main.go -d -f "$(INPUT_FILE)"
