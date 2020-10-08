package crawler

import "github.com/qhzhyt/go-crawler/htmlquery"

// import "htmlquery"

// Response Crawler的响应
type Response struct {
	*htmlquery.Selector
	URL        string
	Status     string
	StatusCode int
	Body       []byte
	Request    *Request
	Method     string
	Headers    map[string]string
	Cookies    map[string]string
	Meta       *map[string]interface{}
}

// NewResponse 创建Response
func NewResponse(content []byte) *Response {
	return &Response{Selector: htmlquery.NewSelector(content), Body: content}
}

// WithRequest 设置request
func (res *Response) WithRequest(req *Request) *Response {
	res.Request = req
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
