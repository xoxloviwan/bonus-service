package main

import (
	"log"
	"net/http"
)

// go build -o ./bin/tmr.exe ./cmd/test_too_many_request/main.go

func main() {

	server := http.Server{
		Addr: ":8090",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			log.Println(req.URL)
			w.Header().Set("Retry-After", "20")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte("No more than 600 requests per minute allowed"))
		}),
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
