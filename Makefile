.PHONY: test run build-lambda clean

test:
	go test ./...

run:
	go run ./cmd/local

build-lambda:
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o bootstrap ./cmd/lambda
	zip prpilot-lambda.zip bootstrap

build-PRPilotFunction:
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o $(ARTIFACTS_DIR)/bootstrap ./cmd/lambda

clean:
	rm -f bootstrap prpilot-lambda.zip
