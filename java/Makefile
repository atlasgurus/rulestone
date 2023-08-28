build: build-darwin-amd64 build-darwin-arm64 build-linux-amd64

build-darwin-amd64:
	@echo "Building for darwin-amd64"
	GOOS=darwin GOARCH=amd64 go build -o ./src/main/resources/rulestone_sidecar_darwin_amd64 ./src/main/go/sidecar.go

build-darwin-arm64:
	@echo "Building for darwin-arm64"
	GOOS=darwin GOARCH=arm64 go build -o ./src/main/resources/rulestone_sidecar_darwin_arm64 ./src/main/go/sidecar.go

build-linux-amd64:
	@echo "Building for linux-amd64"
	GOOS=linux GOARCH=amd64 go build -o ./src/main/resources/rulestone_sidecar_linux_amd64 ./src/main/go/sidecar.go

clean:
	@echo "Cleaning up"
	rm -f ./src/main/resources/rulestone_sidecar_*

.PHONY: build build-darwin-amd64 build-darwin-arm64 build-linux-amd64 clean
