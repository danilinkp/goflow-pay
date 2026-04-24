.PHONY: build-cli
build-cli:
	go build -o ./bin/goflow ./tools/cli/cmd/goflow-cli/main.go

.PHONY: install-cli
install-cli:
	go install ./tools/cli/cmd/goflow-cli