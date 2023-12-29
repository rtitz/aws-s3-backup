# Build it on your own from source

**[Back](../README.md)**

You can just use the pre-compiled binaries in the 'bin/' directory. 
But if you prefer to build it on your own, here is how to do this.

## How to build the executable binary (if you don't like the pre-compiled in the 'bin' directory)
 * [Install Go](https://go.dev/doc/install)
 * Build the binary for your current platform:
```
cd src/
go build -ldflags "-s -w" .
```
 * Build the binary for many platforms:
```
cd src/
go get -u && go mod tidy
bash build.sh  # Not running on Windows
```


## How to execute it directly (without building the binary in advance or using the pre-compiled)
 * [Install Go](https://go.dev/doc/install)
  * Execute:
```
cd src/
go run .
```


**[Back](../README.md)**