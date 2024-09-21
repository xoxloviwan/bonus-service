build:
	go generate ./... && go build -o ./bin/gophermart.exe ./cmd/gophermart/main.go


test:
	go generate ./... && go test ./internal/...

cover:
	go test ./internal/... -coverprofile cover && go tool cover -func cover && rm cover