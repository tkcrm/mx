fmt:
	gofumpt -l -w .

lint:
	golangci-lint run
