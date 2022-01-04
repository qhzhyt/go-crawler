package crawler

import (
	//"bytes"
	"compress/gzip"
	"crypto/x509"
	"github.com/qhzhyt/go-crawler/htmlquery"
	//"golang.org/x/net/html/charset"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

// Cookies Cookies
// type Cookies []*http.Cookie

// import "htmlquery"

type HistoryItem struct {
	Request  *Request
	Response *Response
}

type History []*HistoryItem

// Response Crawler的响应
type Response struct {
	*htmlquery.Selector
	StatusCode int
	URL        string
	Status     string
	Body       []byte
	Request    *Request
	Headers    http.Header
	Cookies    Cookies
	Meta       Meta
	context    *Context
	History    History
	//NativeResponse  *http.Response
	X509Certificate *x509.Certificate
	X509CertChan    []*x509.Certificate
}

// NewResponse 创建Response
// func NewResponse(content []byte) *Response {
// 	return &Response{Selector: htmlquery.NewSelector(content), Body: content}
// }
// NewResponse create a Response from http.Response
func NewResponse(res *http.Response) *Response {
	defer res.Body.Close()
	//res.Request.Body.Close()
	//content, _ := ioutil.ReadAll(res.Body)
	var err error

	var bodyReader io.Reader = res.Body
	//if bodySize > 0 {
	//	bodyReader = io.LimitReader(bodyReader, int64(bodySize))
	//}
	contentEncoding := strings.ToLower(res.Header.Get("Content-Encoding"))
	if !res.Uncompressed && (strings.Contains(contentEncoding, "gzip") ||
		(contentEncoding == "" && strings.Contains(strings.ToLower(res.Header.Get("Content-Type")), "gzip"))) {
		bodyReader, err = gzip.NewReader(bodyReader)
		if err != nil {
			bodyReader = res.Body
		} else {
			defer bodyReader.(*gzip.Reader).Close()
		}
	}
	body, err := ioutil.ReadAll(bodyReader)

	response := &Response{
		//Selector:       htmlquery.NewSelector(body2),
		Body:       body,
		Headers:    res.Header,
		Status:     res.Status,
		StatusCode: res.StatusCode,
		URL:        res.Request.URL.String(),
		History:    History{},
	}

	//res.
	//if len(body) > 0 {
	//	body := bytes.NewReader(content)
	//	r, err := charset.NewReader(body, res.Header.Get("Content-Type"))
	//	if err != nil {
	//		return &Response{Body: content}
	//	}
	//	body2, _ := ioutil.ReadAll(r)
	//	response.Selector = htmlquery.NewSelector(body2)
	//}
	cookies := make(Cookies)
	for _, cookie := range res.Cookies() {
		cookies[cookie.Name] = cookie.Value
	}
	response.Cookies = cookies

	if res.TLS != nil && res.TLS.PeerCertificates != nil && len(res.TLS.PeerCertificates) > 0 {
		response.X509CertChan = res.TLS.PeerCertificates
		for _, cert := range res.TLS.PeerCertificates {
			if cert != nil {
				response.X509Certificate = cert
				break
			}
		}
	}
	return response
}

// WithRequest 设置request
func (res *Response) WithRequest(req *Request) *Response {
	res.Request = req
	res.Meta = req.Meta
	res.context = req.context
	res.History = req.History
	if res.context.Settings.AutoParseHtml {
		res.Selector = htmlquery.NewSelector(res.Body)
	}
	return res
}

// WithStatus 设置响应状态
func (res *Response) WithStatus(code int, sataus string) *Response {
	res.StatusCode = code
	res.Status = sataus
	return res
}

func (res *Response) Redirect(url string) *Request {
	req := res.Request.Clone()
	req.History = res.History.Append(res.Request, res)
	//req.History = append(res.History, &HistoryItem{Request: req, Response: res})
	//u, _ := urlLib.Parse(res.Request.URL)
	req.URL = URLJoin(res.Request.URL, url)
	//req.
	req.redirectTimes++
	return req
}

// // Selector Selector
// type Selector struct {
// }

// Text 获取响应文本
func (res *Response) Text() string {
	return string(res.Body)
}

func (h History) Append(request *Request, response *Response) History {
	his := History{}
	copy(his, h)
	return append(his, &HistoryItem{Request: request, Response: response})
}
