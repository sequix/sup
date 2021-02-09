build:
	GOOS=linux GOARCH=amd64 go build -o sup -trimpath -ldflags "-s -w -X 'github.com/sequix/sup/pkg/buildinfo.Commit=$(git rev-parse HEAD)'" cmd/main.go

compress: build
	upx -9 -f -q ./sup

clean:
	rm -f ./sup