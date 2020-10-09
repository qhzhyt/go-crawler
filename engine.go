package crawler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"sync/atomic"
	"time"
)

// CrawlEngine 爬取引擎
type CrawlEngine struct {
	// context             *Context
	crawler             *Crawler
	cookieJar           *http.CookieJar
	visitedUrls         map[string]bool
	httpClient          *http.Client
	RequestQueue        chan *Request
	ItemQueue           chan *itemWrapper
	RequestingCount     int32
	ProcessingItemCount int32
	Settings            *Settings
	RequestMetaMap      map[*http.Request]Meta
}

type itemWrapper struct {
	item    interface{}
	context *Context
}

// Start 启动引擎
func (eng *CrawlEngine) Start() {
	go eng.StartProcessRequests()
	go eng.StartProcessItems()
}

func proxyFunc(eng *CrawlEngine) func(req *http.Request) (*url.URL, error) {

	return func(req *http.Request) (*url.URL, error) {
		meta := eng.RequestMetaMap[req]
		proxy := meta["ProxyURL"]
		if proxy != nil && proxy.(string) != "" {
			return url.Parse(proxy.(string))
		}
		return nil, nil
	}
}

func afterRequestFunc(eng *CrawlEngine) func(req *http.Request) {
	return func(req *http.Request) {
		fmt.Println(eng.RequestMetaMap)
	}
}

// NewCrawlerEngine NewCrawlerEngine
func newCrawlerEngine(settings *Settings) *CrawlEngine {
	eng := &CrawlEngine{
		Settings:       settings,
		RequestQueue:   make(chan *Request, 100),
		ItemQueue:      make(chan *itemWrapper, 100),
		RequestMetaMap: make(map[*http.Request]Meta),
	}

	httpClient := &http.Client{
		Transport: &http.Transport{Proxy: proxyFunc(eng)},
	}

	eng.httpClient = httpClient

	return eng
}

// StartProcessItems 开始处理Items
func (eng *CrawlEngine) StartProcessItems() {
	workerCount := int32(6)
	if eng.Settings.MaxConcurrentProcessItems > 0 {
		workerCount = eng.Settings.MaxConcurrentProcessItems
	}

	worker := func() {
		hasInc := false
		defer func() {
			if hasInc {
				atomic.AddInt32(&eng.ProcessingItemCount, -1)
			}
		}()

		for itemW := range eng.ItemQueue {
			item := itemW.item
			ctx := itemW.context
			atomic.AddInt32(&eng.ProcessingItemCount, 1)
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
			atomic.AddInt32(&eng.ProcessingItemCount, -1)
			hasInc = false
		}
	}

	for i := int32(0); i < workerCount; i++ {
		go worker()
	}
}

// StartProcessRequests 开始处理请求
func (eng *CrawlEngine) StartProcessRequests() {
	// request = http.Request{Method: "GET"}
	// request.
	fmt.Println(eng)
	worker := func(req *Request) {
		ctx := req.context
		atomic.AddInt32(&eng.RequestingCount, 1)
		defer atomic.AddInt32(&eng.RequestingCount, -1)

		timeout := time.Duration(20 * time.Microsecond)
		if req.Timeout != 0 {
			timeout = time.Duration(req.Timeout) * time.Microsecond
		}
		_context, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		request := req.toHTTPRequest()
		// runtime.SetFinalizer(request, afterRequestFunc(eng))
		if len(req.Meta) > 0 {
			eng.RequestMetaMap[request] = req.Meta
			defer delete(eng.RequestMetaMap, request)
		}

		if request == nil {
			fmt.Println("request is nil")
		}
		request.WithContext(_context)

		response, err := eng.httpClient.Do(request)
		if err != nil {
			fmt.Println(err)
		}

		if response == nil {
			return
		}

		// response.Header
		res := NewResponse(response).WithRequest(req)

		ctx.LastResponse = res
		// &Response{Body: body, Status: response.Status, StatusCode: response.StatusCode, Request: req}
		if req.Callback != nil {
			req.Callback(res, ctx)
		} else if eng.crawler.RequestCallback != nil {
			eng.crawler.RequestCallback(res, ctx)
		}
	}
	for req := range eng.RequestQueue {
		fmt.Println(req)
		// if req.URL == "action::stop" {
		// 	break
		// }
		if eng.Settings.RequestDelay > 0 {
			worker(req)
			time.Sleep(time.Duration(eng.Settings.RequestDelay) * time.Millisecond)
		} else {
			for eng.RequestingCount >= eng.Settings.MaxConcurrentRequests {
				time.Sleep(time.Millisecond * 200)
			}

			go worker(req)

		}
	}

	// httpClient.Do(&http.Request{Method: "GET"})
}

// Wait 等待引擎执行结束
func (eng *CrawlEngine) Wait() {
	// ctx := eng.context
	interval := time.Millisecond * 200
	for !(eng.IsIdle()) {
		time.Sleep(interval)
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
	return eng.RequestingCount > 0 || len(eng.RequestQueue) > 0
}

// isProcessingItems 判断是否有Item正在处理, 或者还未处理
func (eng *CrawlEngine) isProcessingItems() bool {
	return eng.ProcessingItemCount > 0 || len(eng.ItemQueue) > 0
}
