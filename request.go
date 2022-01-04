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

type RedirectCallback func(res *Response, req *Request, ctx *Context) *Request

// Request Crawler的请求
type Request struct {
	Method        string
	URL           string
	Body          []byte
	Headers       http.Header
	Cookies       Cookies
	Timeout       int
	Meta          Meta
	Callback      ResponseCallback
	ErrorCallback RequestErrorCallback
	context       *Context
	ProxyURL      string
	OriginURL     string
	Host          string
	History       History
	retryTimes    int
	redirectTimes int
}

// Args is http post form
type Args url.Values

func (req *Request) toHTTPRequest() (*http.Request, error) {
	// url, _ := urlLib.Parse(req.URL)
	// rc, ok := body.(io.ReadCloser)
	// if !ok && body != nil {
	// 	rc = ioutil.NopCloser(body)
	// }
	result, err := http.NewRequest(req.Method, req.URL, bytes.NewReader(req.Body))
	if err != nil {
		// log.Println(err)
		return nil, err
	}
	//for k, v := range req.Headers {
	//	if v != nil {
	//		switch v.(type) {
	//		case string:
	//			result.Header.Set(k, v.(string))
	//		case []string:
	//			values := v.([]string)
	//			if values != nil && len(values) > 0 {
	//				result.Header.Set(k, values[0])
	//			}
	//		}
	//	}
	//}
	result.Header = req.Headers

	for k, v := range req.Cookies {
		if v != "" {
			result.AddCookie(&http.Cookie{Name: k, Value: v})
		}
	}

	if req.ProxyURL != "" {
		req.Meta["ProxyURL"] = req.ProxyURL
	}

	if req.Host != "" {
		result.Host = req.Host
	}

	return result, nil
}

// NewRequest NewRequest
func NewRequest(method string, url string, body []byte) *Request {
	return &Request{Method: method, URL: url, Body: body, Headers: make(http.Header), Cookies: make(Cookies), Meta: make(Meta), OriginURL: url}
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

	for i, url0 := range urls {
		result[i] = GetURL(url0)
	}

	return result
}

// WithContentType set Content-Type
func (req *Request) WithContentType(contentType string) *Request {
	req.Headers.Set("Content-Type", contentType)
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
		req.Headers.Set(k, v)
	}
	return req
}

// WithHeaders set Headers
func (req *Request) AddHeader(key string, value string) *Request {
	req.Headers.Add(key, value)
	return req
}

// WithMeta set Headers
func (req *Request) WithMeta(meta Meta) *Request {
	req.Meta = meta
	return req
}

// AddMeta set Headers
func (req *Request) AddMeta(key string, value interface{}) *Request {
	if req.Meta != nil {
		req.Meta[key] = value
	}
	return req
}

// WithProxy set Headers
func (req *Request) WithProxy(proxy string) *Request {
	req.ProxyURL = proxy
	return req
}

// WithHost set Host
func (req *Request) WithHost(host string) *Request {
	req.Host = host
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

func (req *Request) Clone() *Request {
	return &Request{
		Method:        req.Method,
		URL:           req.URL,
		Headers:       req.Headers,
		Cookies:       req.Cookies,
		Body:          req.Body,
		Timeout:       req.Timeout,
		Callback:      req.Callback,
		ErrorCallback: req.ErrorCallback,
		Meta:          req.Meta,
		ProxyURL:      req.ProxyURL,
		OriginURL:     req.OriginURL,
		context:       req.context,
		redirectTimes: req.redirectTimes,
	}
}

func (m Meta) Has(key string) bool {
	return m[key] != nil
}

//// Get a header value by name
//func (h Headers) Get(name string) string {
//	values := h[name]
//	switch values.(type) {
//	case string:
//		return values.(string)
//	case []string:
//
//		vs := values.([]string)
//		if vs != nil && len(vs) > 0 {
//			return vs[0]
//		}
//	}
//	return ""
//}
//
//// Set a header value by name
//func (h Headers) Set(name string, value string) {
//	h[name] = value
//}
//
//func (h Headers) GetList(name string) []string {
//	values := h[name]
//	switch values.(type) {
//	case string:
//		return []string{values.(string)}
//	case []string:
//
//		return values.([]string)
//
//	}
//	return []string{}
//}
