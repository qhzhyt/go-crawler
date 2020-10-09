package crawler

import (
	"bytes"
	"io/ioutil"
	"net/http"

	"github.com/qhzhyt/go-crawler/htmlquery"
	"golang.org/x/net/html/charset"
)

// Cookies Cookies
// type Cookies []*http.Cookie

// import "htmlquery"

// Response Crawler的响应
type Response struct {
	StatusCode int
	*htmlquery.Selector
	URL            string
	Status         string
	Body           []byte
	Request        *Request
	Headers        Headers
	Cookies        Cookies
	Meta           Meta
	context        *Context
	NativeResponse *http.Response
}

// NewResponse 创建Response
// func NewResponse(content []byte) *Response {
// 	return &Response{Selector: htmlquery.NewSelector(content), Body: content}
// }
// NewResponse create a Response from http.Response
func NewResponse(res *http.Response) *Response {
	defer res.Body.Close()
	content, _ := ioutil.ReadAll(res.Body)

	body := bytes.NewReader(content)

	r, err := charset.NewReader(body, res.Header.Get("Content-Type"))
	if err != nil {
		return &Response{Body: content}
	}
	body2, _ := ioutil.ReadAll(r)

	cookies := make(Cookies)
	for _, cookie := range res.Cookies() {
		cookies[cookie.Name] = cookie.Value
	}

	headers := make(Headers)
	for key, value := range res.Header {
		if len(value) > 0 {
			if len(value) == 0 {
				headers[key] = value[0]
			} else {
				headers[key] = value
			}
		}
	}
	return &Response{
		Selector:       htmlquery.NewSelector(body2),
		Body:           content,
		Headers:        headers,
		Status:         res.Status,
		StatusCode:     res.StatusCode,
		Cookies:        cookies,
		NativeResponse: res,
		URL:            res.Request.URL.String(),
	}
}

// WithRequest 设置request
func (res *Response) WithRequest(req *Request) *Response {
	res.Request = req
	res.Meta = req.Meta
	res.context = req.context
	return res
}

// WithStatus 设置响应状态
func (res *Response) WithStatus(code int, sataus string) *Response {
	res.StatusCode = code
	res.Status = sataus
	return res
}

// // Selector Selector
// type Selector struct {
// }

// Text 获取响应文本
func (res *Response) Text() string {
	return string(res.Body)
}
