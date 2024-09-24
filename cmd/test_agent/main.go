package main

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"

	luhn "github.com/ShiraazMoollatjie/goluhn"
	"github.com/go-resty/resty/v2"
)

//  go build -o ./bin/order_agent.exe ./cmd/test_agent/main.go

func Generate(length int) string {

	var s strings.Builder
	for i := 0; i < length-1; i++ {
		s.WriteString(strconv.Itoa(rand.Intn(9)))
	}

	_, res, _ := luhn.Calculate(s.String()) //ignore error because this will always be valid
	return res
}

func main() {
	wg := sync.WaitGroup{}
	cnt := 100
	wg.Add(cnt)
	client := resty.New()
	for i := 0; i < cnt; i++ {
		orderID := Generate(10)
		go func() {
			defer wg.Done()
			resp, err := client.R().
				SetBody(orderID).
				SetAuthToken("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3MjYyMTMzNzgsIlVzZXJJRCI6MX0.yUTYgftFBf6mBwx839O77doZXmRPqVlS-H05oRrbOtA").
				Post("http://localhost:8080/api/user/orders")
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Println(orderID, resp.StatusCode())
		}()
	}
	wg.Wait()
}
