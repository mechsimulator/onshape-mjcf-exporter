package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

var StlUrlRedirects []string

func CheckRedirectFunc(req *http.Request, via []*http.Request) error {
	StlUrlRedirects = append(StlUrlRedirects, req.URL.String())
	return http.ErrAbortHandler
}

func StlRequest(client *http.Client, ctx context.Context, url string, auth string) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		fmt.Println("failed to make stl request:", err.Error())
	}
	req.Header.Add("accept", "application/octet-stream")
	req.Header.Add("Authorization", auth)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("request failed:", err.Error())
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("failed to read stl response body:", err.Error())
	}
	fmt.Println(string(body))
}
