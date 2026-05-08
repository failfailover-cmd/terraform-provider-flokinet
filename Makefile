default: build

NAME=flokinet
BINARY=terraform-provider-flokinet

build:
	go build -o $(BINARY)

test:
	go test -v ./...

clean:
	rm -f $(BINARY)

release-snapshot:
	goreleaser release --snapshot --clean
