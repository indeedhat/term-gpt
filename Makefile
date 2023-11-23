.PHONY: build
build:
	go build -o build/term-gpt ./cmd/term-gpt/main.go

.PHONY: run
run:
	go run  ./cmd/term-gpt/main.go
