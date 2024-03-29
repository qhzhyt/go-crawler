package crawler

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"sync"
	"time"
)

// CrawlEngine 爬取引擎
type CrawlEngine struct {
	// context             *Context
	crawler     *Crawler
	cookieJar   *http.CookieJar
	visitedUrls map[string]bool
	httpClient  *http.Client
	//fastHttpClient *fasthttp.Client
	RequestQueue chan *Request
	ItemQueue    chan *itemWrapper
	//RequestingCount     int32
	//ProcessingItemCount int32
	Settings           *Settings
	RequestMetaMap     *sync.Map //map[*http.Request]Meta
	requestingChan     chan *Request
	processingItemChan chan bool
}

type itemWrapper struct {
	item    interface{}
	context *Context
}

// Start 启动引擎
func (eng *CrawlEngine) Start() {
	eng.requestingChan = make(chan *Request, eng.Settings.MaxConcurrentRequests)
	eng.processingItemChan = make(chan bool, eng.Settings.MaxConcurrentProcessItems)
	go eng.StartProcessRequests()
	go eng.StartProcessItems()
}

func proxyFunc(eng *CrawlEngine) func(req *http.Request) (*url.URL, error) {

	return func(req *http.Request) (*url.URL, error) {
		meta, status := eng.RequestMetaMap.Load(req)
		if status {
			proxy := meta.(Meta)["ProxyURL"]
			if proxy != nil && proxy.(string) != "" {
				return url.Parse(proxy.(string))
			}
		}
		return nil, nil
	}
}

func checkRedirect(req *http.Request, via []*http.Request) error {
	return http.ErrUseLastResponse
}

func afterRequestFunc(eng *CrawlEngine) func(req *http.Request) {
	return func(req *http.Request) {
		fmt.Println(eng.RequestMetaMap)
	}
}

func creatHttpClient(transport *http.Transport, engine *CrawlEngine) *http.Client {
	if transport == nil {
		transport = &http.Transport{
			ResponseHeaderTimeout: 20 * time.Second,
			IdleConnTimeout:       20 * time.Second,
			TLSHandshakeTimeout:   20 * time.Second,
			DialContext: (&net.Dialer{
				Timeout: 20 * time.Second,
			}).DialContext,
		}
	}

	transport.MaxIdleConnsPerHost = 1
	transport.DisableKeepAlives = true
	transport.MaxIdleConns = 1
	transport.Proxy = proxyFunc(engine)
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: engine.Settings.SkipTLSVerify}

	if transport.ResponseHeaderTimeout == 0 {
		transport.ResponseHeaderTimeout = 20 * time.Second
	}
	if transport.IdleConnTimeout == 0 {
		transport.IdleConnTimeout = 20 * time.Second
	}
	if transport.TLSHandshakeTimeout == 0 {
		transport.TLSHandshakeTimeout = 20 * time.Second
	}

	return &http.Client{
		CheckRedirect: checkRedirect,
		Transport:     transport,
		Timeout:       time.Second * 60,
	}

}

// NewCrawlerEngine NewCrawlerEngine
func newCrawlerEngine(settings *Settings) *CrawlEngine {
	fmt.Println(settings)
	eng := &CrawlEngine{
		Settings:     settings,
		RequestQueue: make(chan *Request, settings.MaxConcurrentRequests*3),
		ItemQueue:    make(chan *itemWrapper, 1000),
		//requestingChan: make(chan *Request, settings.MaxConcurrentRequests),
		RequestMetaMap: &sync.Map{},
	}

	eng.httpClient = creatHttpClient(settings.Transport, eng)

	//eng.fastHttpClient = &fasthttp.D

	return eng
}

// StartProcessItems 开始处理Items
func (eng *CrawlEngine) StartProcessItems() {
	workerCount := 6
	if eng.Settings.MaxConcurrentProcessItems > 0 {
		workerCount = eng.Settings.MaxConcurrentProcessItems
	}

	worker := func() {
		hasInc := false

		defer func() {
			if hasInc {
				<-eng.processingItemChan
			}
		}()

		for itemW := range eng.ItemQueue {
			item := itemW.item
			ctx := itemW.context
			eng.processingItemChan <- true
			hasInc = true

			/* pipeline 执行顺序
			** 1）先执行指定类型的处理函数
			** 2）若上步返回值非空则继续执行通用处理函数
			** 3）若上步返回值非空继续按顺序执行pipeline列表中的pipeline
			 */

			if item != nil && len(eng.crawler.ItemTypeFuncs) > 0 {
				if len(eng.crawler.ItemTypeFuncs) > 1 || eng.crawler.ItemTypeFuncs["*"] == nil {
					itemType := reflect.TypeOf(item).String()
					if processFunc := eng.crawler.ItemTypeFuncs[itemType]; processFunc != nil {
						item = processFunc(item, ctx)
					}
				}

				if processFunc := ctx.Crawler.ItemTypeFuncs["*"]; processFunc != nil && item != nil {
					item = processFunc(item, ctx)
				}
			}

			if item != nil && ctx.Crawler.Pipelines != nil && len(ctx.Crawler.Pipelines) > 0 {
				for _, pipeline := range ctx.Crawler.Pipelines {
					newItem := pipeline.ProcessItem(item, ctx)
					if newItem != nil {
						item = newItem
					}
				}
			}
			<-eng.processingItemChan
			//atomic.AddInt32(&eng.ProcessingItemCount, -1)
			hasInc = false
		}
	}

	for i := 0; i < workerCount; i++ {
		go worker()
	}
}

func (eng *CrawlEngine) processResponseCallback(req *Request, res *Response) {
	req.context.LastResponse = res
	// &Response{Body: body, Status: response.Status, StatusCode: response.StatusCode, Request: req}
	if req.Callback != nil {
		req.Callback(res, req.context)
	}
	if eng.crawler.responseCallback != nil {
		eng.crawler.responseCallback(res, req.context)
	}
}

func (eng CrawlEngine) processRequestErrorCallback(req *Request, err error) {
	if req.ErrorCallback != nil {
		req.ErrorCallback(req, err, req.context)
	}
	if eng.crawler.requestErrorCallback != nil {
		eng.crawler.requestErrorCallback(req, err, req.context)
	}
}

// StartProcessRequests 开始处理请求
func (eng *CrawlEngine) StartProcessRequests() {

	worker := func(req *Request) *Request {
		defer func() {
			<-eng.requestingChan
		}()
		ctx := req.context
		//atomic.AddInt32(&eng.RequestingCount, 1)
		//defer atomic.AddInt32(&eng.RequestingCount, -1)

		//timeout := 20 * time.Microsecond
		//if req.Timeout != 0 {
		//	timeout = time.Duration(req.Timeout) * time.Microsecond
		//}
		//_context, cancel := context.WithTimeout(context.Background(), timeout)
		//defer cancel()
		request, err := req.toHTTPRequest()
		// runtime.SetFinalizer(request, afterRequestFunc(eng))

		if request == nil {
			fmt.Println("request is nil")
			eng.processRequestErrorCallback(req, err)
			return req
		}

		if len(req.Meta) > 0 {
			//eng.RequestMetaMap[request] = req.Meta
			eng.RequestMetaMap.Store(request, req.Meta)
			defer eng.RequestMetaMap.Delete(request)
		}
		//request.WithContext(_context)

		response, err := eng.httpClient.Do(request)

		if response == nil {
			//if req == nil {
			//	req.Meta["retryTimes"] = 0
			//}

			if req.retryTimes < eng.Settings.MaxRetryTimes {
				req.retryTimes++
				go ctx.retry(req)
			} else {
				eng.processRequestErrorCallback(req, err)
			}
			//return
		} else {
			// response.Header

			res := NewResponse(response).WithRequest(req)

			//fmt.Println(res.StatusCode)

			if res.StatusCode == 301 || res.StatusCode == 302 || res.StatusCode == 303 || res.StatusCode == 307 {
				//	处理重定向
				redirectUrl := res.Headers.Get("Location")

				newReq := res.Redirect(redirectUrl)

				if newReq.redirectTimes > eng.Settings.MaxRedirectTimes {
					eng.processRequestErrorCallback(req, errors.New(fmt.Sprintf("redirect too many times: %d", newReq.redirectTimes)))
				} else {
					if eng.crawler.redirectCallback != nil {
						//newReq := res.Redirect()
						result := eng.crawler.redirectCallback(res, newReq, ctx)

						if result != nil {
							eng.RequestQueue <- result
						} else {
							//eng.processResponseCallback(req, res)
						}
					} else {
						//redirectUrl := res.Headers.Get("Location")
						eng.RequestQueue <- newReq
					}
				}

			} else {
				eng.processResponseCallback(req, res)
			}
		}

		//fmt.Println(len(eng.requestingChan))

		return req
	}
	for req := range eng.RequestQueue {
		//fmt.Println(req)
		// if req.URL == "action::stop" {
		// 	break
		//}
		//fmt.Println(req)
		if eng.Settings.RequestDelay > 0 {
			worker(req)
			time.Sleep(time.Duration(eng.Settings.RequestDelay) * time.Millisecond)
		} else {
			//for eng.RequestingCount >= eng.Settings.MaxConcurrentRequests {
			//	time.Sleep(time.Millisecond * 200)
			//}
			eng.requestingChan <- req
			go worker(req)
		}
	}

	// httpClient.Do(&http.Request{Method: "GET"})
}

// Wait 等待引擎执行结束
func (eng *CrawlEngine) Wait() {
	// ctx := eng.context
	interval := time.Millisecond * 1000
	for !(eng.IsIdle()) {
		time.Sleep(interval)
	}
}

// Wait 等待引擎执行结束
func (eng *CrawlEngine) WaitTime(seconds time.Duration) {
	// ctx := eng.context
	interval := time.Millisecond * 1000
	for !(eng.IsIdle()) && seconds > 0 {
		time.Sleep(interval)
		seconds--
	}
}

// IsIdle 判断引擎是否进入空闲状态
func (eng *CrawlEngine) IsIdle() bool {
	return !eng.isProcessingRequests() && !eng.isProcessingItems()
}

func (eng *CrawlEngine) doRequest(u, method string, depth int, requestData io.Reader, ctx *Context, hdr http.Header, req *http.Request) error {

	// defer c.wg.Done()
	// if ctx == nil {
	// 	ctx = NewContext()
	// }
	// request := &Request{
	// 	URL:       req.URL,
	// 	Headers:   &req.Header,
	// 	Ctx:       ctx,
	// 	Depth:     depth,
	// 	Method:    method,
	// 	Body:      requestData,
	// 	collector: c, // 这里将Collector放到request中，这个可以对请求继续处理
	// 	ID:        atomic.AddUint32(&c.requestCount, 1),
	// }
	// // 回调函数处理 request
	// c.handleOnRequest(request)

	// if request.abort {
	// 	return nil
	// }

	// if method == "POST" && req.Header.Get("Content-Type") == "" {
	// 	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	// }

	// if req.Header.Get("Accept") == "" {
	// 	req.Header.Set("Accept", "*/*")
	// }

	// origURL := req.URL
	// // 这里是 去请求网络， 是调用了 `http.Client.Do`方法请求的
	// response, err := c.backend.Cache(req, c.MaxBodySize, c.CacheDir)
	// if proxyURL, ok := req.Context().Value(ProxyURLKey).(string); ok {
	// 	request.ProxyURL = proxyURL
	// }
	// // 回调函数，处理error
	// if err := c.handleOnError(response, err, request, ctx); err != nil {
	// 	return err
	// }
	// if req.URL != origURL {
	// 	request.URL = req.URL
	// 	request.Headers = &req.Header
	// }
	// atomic.AddUint32(&c.responseCount, 1)
	// response.Ctx = ctx
	// response.Request = request

	// err = response.fixCharset(c.DetectCharset, request.ResponseCharacterEncoding)
	// if err != nil {
	// 	return err
	// }
	// // 回调函数 处理Response
	// c.handleOnResponse(response)

	// // 回调函数 HTML
	// err = c.handleOnHTML(response)
	// if err != nil {
	// 	c.handleOnError(response, err, request, ctx)
	// }
	// // 回调函数XML
	// err = c.handleOnXML(response)
	// if err != nil {
	// 	c.handleOnError(response, err, request, ctx)
	// }
	// // 回调函数 Scraped
	// c.handleOnScraped(response)

	return nil
}

// isProcessingRequests 判断是否有请求正在处理, 或者还未处理
func (eng *CrawlEngine) isProcessingRequests() bool {
	return len(eng.requestingChan) > 0 || len(eng.RequestQueue) > 0
}

// isProcessingItems 判断是否有Item正在处理, 或者还未处理
func (eng *CrawlEngine) isProcessingItems() bool {
	return len(eng.processingItemChan) > 0 || len(eng.ItemQueue) > 0
}
