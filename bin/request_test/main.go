package main

import (
	"fmt"
	"io"
	"net/http"
)

func resp_simple() {
	resp, err := http.Get("http://127.0.0.1:5000/v1/model/list")
	fmt.Println(resp, "\n", err)
}

func resp_ex() {
	//resp, err := http.NewRequest("GET", "http://localhost:5000/v1/model/list")
}

func main() {
	resp, err := http.Get("http://localhost:5000/v1/internal/model/list")
	fmt.Println(resp, "\n", err)
	if err == nil && resp.StatusCode == 200 {
		val, err := io.ReadAll(resp.Body)
		if err == nil {
			fmt.Println(string(val))
		}
	}
}
