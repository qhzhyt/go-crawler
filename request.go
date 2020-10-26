package crawler

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	urlLib "net/url"
)

// Headers request or response headers
type Headers map[string]interface{}

// Cookies request or response Cookies
type Cookies map[string]string

// Meta request or response Meta
type Meta map[string]interface{}

// ResponseCallback ResponseCallback
type ResponseCallback func(res *Response, ctx *Context)

type RequestErrorCallback func(req *Request, err error, ctx *Context)

// Request Crawler的请求
type Request struct {
	Method        string
	URL           string
	Body          []byte
	Headers       Headers
	Cookies       Cookies
	Timeout       int
	Meta          Meta
	Callback      ResponseCallback
	ErrorCallback RequestErrorCallback
	context       *Context
	ProxyURL      string
}

// Args is http post form
type Args url.Values

func (req *Request) toHTTPRequest() *http.Request {
	// url, _ := urlLib.Parse(req.URL)
	// rc, ok := body.(io.ReadCloser)
	// if !ok && body != nil {
	// 	rc = ioutil.NopCloser(body)
	// }
	result, err := http.NewRequest(req.Method, req.URL, bytes.NewReader(req.Body))
	if err != nil {
		return nil
	}
	for k, v := range req.Headers {
		if v != nil {
			switch v.(type) {
			case string:
				result.Header.Set(k, v.(string))
			case []string:
				values := v.([]string)
				if values != nil && len(values) > 0 {
					result.Header.Set(k, values[0])
				}
			}
		}
	}

	for k, v := range req.Cookies {
		if v != "" {
			result.AddCookie(&http.Cookie{Name: k, Value: v})
		}
	}

	if req.ProxyURL != "" {
		req.Meta["ProxyURL"] = req.ProxyURL
	}

	return result
}

// NewRequest NewRequest
func NewRequest(method string, url string, body []byte) *Request {
	return &Request{Method: method, URL: url, Body: body, Headers: make(Headers), Cookies: make(Cookies), Meta: make(Meta)}
}

// GetRequest GetRequest
func GetRequest(url string, args Args) *Request {
	argString := urlLib.Values(args).Encode()
	if argString != "" {
		url += "?" + argString
	}

	return GetURL(url)
}

// PostRequest basic post request
func PostRequest(url string, body []byte) *Request {
	return NewRequest("POST", url, body)
}

// FormRequest form post request
func FormRequest(url string, form Args) *Request {
	return PostRequest(url, []byte(urlLib.Values(form).Encode())).WithContentType("application/x-www-form-urlencoded")
}

// GetURL GET url
func GetURL(url string) *Request {
	return NewRequest("GET", url, nil)
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

// WithContentType set Content-Type
func (req *Request) WithContentType(contentType string) *Request {
	req.Headers["Content-Type"] = contentType
	return req
}

// WithTimeout set timeout
func (req *Request) WithTimeout(timeout int) *Request {
	req.Timeout = timeout
	return req
}

// WithHeaders set Headers
func (req *Request) WithHeaders(headers map[string]string) *Request {
	for k, v := range headers {
		req.Headers[k] = v
	}
	return req
}

// WithMeta set Headers
func (req *Request) WithMeta(meta Meta) *Request {
	req.Meta = meta
	return req
}

// WithProxy set Headers
func (req *Request) WithProxy(proxy string) *Request {
	req.ProxyURL = proxy
	return req
}

// WithCookies set Cookies
func (req *Request) WithCookies(cookies map[string]string) *Request {
	for k, v := range cookies {
		req.Cookies[k] = v
	}
	return req
}

// OnResponse set Response callback
func (req *Request) OnResponse(callback ResponseCallback) *Request {
	req.Callback = callback
	return req
}

func (req *Request) OnError(callback RequestErrorCallback) *Request {
	req.ErrorCallback = callback
	return req
}

// Get a header value by name
func (h Headers) Get(name string) string {
	values := h[name]
	switch values.(type) {
	case string:
		return values.(string)
	case []string:

		vs := values.([]string)
		if vs != nil && len(vs) > 0 {
			return vs[0]
		}
	}
	return ""
}

// Set a header value by name
func (h Headers) Set(name string, value string) {
	h[name] = value
}

func (h Headers) GetList(name string) []string {
	values := h[name]
	switch values.(type) {
	case string:
		return []string{values.(string)}
	case []string:

		return values.([]string)

	}
	return []string{}
}
