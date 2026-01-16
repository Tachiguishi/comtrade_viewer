# go test -bench=./test
go test -v ./test -coverprofile=coverage.out -cover -coverpkg=./... -coverprofile=coverage.out && go tool cover -html=coverage.out -o coverage.html && rm coverage.out
