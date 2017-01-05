
clean:
	go clean -i .
	rm -rf vendor

prep:
	@which glide || { echo "glide: command not found!"; exit 1; }

test:
	go test -cover .

cov:
	go test -coverprofile=/tmp/coverage.out
	go tool cover -html=/tmp/coverage.out

