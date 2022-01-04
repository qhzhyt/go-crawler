package crawler

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestCrawler(t *testing.T) {

	onResponse := func(res *Response, ctx *Context) {
		t.Error(res.Text())
	}

	onRedirect := func(response *Response, request *Request, ctx *Context) *Request {

		fmt.Println("redirect to ", request.URL)
		fmt.Println("history ", request.History, response.History)
		return request
	}

	onError := func(req *Request, err error, ctx *Context) {
		t.Error(err)
	}

	settings := &Settings{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			DialTLS: func(network, addr string) (net.Conn, error) {
				//fmt.Println(ctx.Value("ip"), network, addr)
				host := addr[:strings.Index(addr, ":")]
				if strings.Contains(addr, "--with-ip--") {
					host = strings.Split(addr, "--with-ip--")[0]
					addr = strings.Split(addr, "--with-ip--")[1]
				}

				dialer := &net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: 30 * time.Second,
					DualStack: true,
				}
				conn, err := tls.DialWithDialer(dialer, network, addr, &tls.Config{
					InsecureSkipVerify: false,
					ServerName:         host,
				})
				if err != nil {
					return conn, err
				}
				return conn, nil
			},
		},
	}

	a := NewCrawler(settings).OnResponse(onResponse).OnRedirect(onRedirect).OnRequestError(onError)
	a.CrawlURL("https://www.abaidu.com--with-ip--220.181.38.149")
	a.Start(true)
	t.Log("hello world")
}
