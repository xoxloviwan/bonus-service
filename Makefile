build:
	go build -o ./bin/gophermart.exe ./cmd/gophermart/main.go

mock:
	mockgen -destination ./internal/mock/store_mock.go -package mock gophermart/internal/api Store
	mockgen -destination ./internal/polling/store_mock.go -package polling gophermart/internal/polling Store
	mockgen -destination ./internal/api/poller_mock.go -package api gophermart/internal/api Poller

test:
	go test -v ./internal/...

cover:
	go test ./internal/... -coverprofile cover && go tool cover -func cover && rm cover