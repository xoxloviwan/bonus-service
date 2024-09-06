build:
	go build -o ./bin/gophermart.exe ./cmd/gophermart/main.go

mock:
	mockgen -destination ./internal/mock/store_mock.go -package mock gophermart/internal/api Store

test:
	go test -v ./internal/api

cover:
	go test ./internal/api -coverprofile cover && go tool cover -func cover && rm cover