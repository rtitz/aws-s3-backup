# 🔨 Build it on your own from source

**[Back](../README.md)**

You can just use the pre-compiled binaries in the 'bin/' directory. 
But if you prefer to build it on your own, here is how to do this.

## 🛠️ How to build the executable binary (if you don't like the pre-compiled in the 'bin' directory)
 * 📦 [Install Go](https://go.dev/doc/install)
 * 🔨 Build the binary for your current platform:
```
cd src/
go build -o ../bin/aws-s3-backup .
```
 * 🌍 Build the binary for many platforms:
```
cd src/
go mod tidy
bash build.sh  # Not running on Windows
```
   * 📋 **build.sh** cross-compiles for multiple platforms:
     * **Enabled**: macOS (arm64/amd64), Linux (386/amd64/arm/arm64), Windows (386/amd64/arm/arm64)
     * **Available**: FreeBSD, NetBSD, OpenBSD, RISC-V, PowerPC64 (uncomment in build.sh)
     * **Custom architectures**: Edit the `platforms` array in build.sh to enable additional targets

## 🧪 How to run tests
 * ▶️ Run all tests:
```
cd src/
go test ./...
```
 * 📈 Run tests with coverage:
```
cd src/
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```
 * 🔍 Run specific test categories:
```
cd src/
go test ./tests/          # Unit tests
go test ./utils/          # Utility tests
go test ./config/         # Configuration tests
```

## ▶️ How to execute it directly (without building the binary in advance or using the pre-compiled)
 * 📦 [Install Go](https://go.dev/doc/install)
 * ▶️ Execute:
```
cd src/
go run . -help
```
 * 📋 Test with dry-run:
```
cd src/
# Test backup dry-run
go run . -json ../example-input.json -dryrun

# Test restore dry-run (using local directory)
go run . -mode restore -bucket /path/to/local/backup -destination /tmp/restore -dryrun
```

**[Back](../README.md)**