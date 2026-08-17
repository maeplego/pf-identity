.PHONY: test run

test:
	go test ./...

run:
	IDENTITY_DEV_GENERATE_KEYS=true IDENTITY_STORE=memory go run ./cmd/server
