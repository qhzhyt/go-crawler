package crawler

import (
	"fmt"
)

// Request Crawler的请求
type Request struct {
	URL      string
	Method   string
	Body     []byte
	Headers  map[string]string
	Cookies  map[string]string
	Timeout  int
	Meta     *map[string]interface{}
	Callback func(res *Response, ctx *Context)
}

// GetURL GET url
func GetURL(url string) *Request {
	return &Request{URL: url, Method: "GET"}
}

// GetURLs GET url
func GetURLs(urls ...string) []*Request {
	fmt.Println()

	result := make([]*Request, len(urls))

	for i, url := range urls {
		result[i] = GetURL(url)
	}

	return result
}
