.PHONY: test
test:
	@go test -v -cover ./...

.PHONY: update
update:
	@go get -u
	@go mod tidy
