package crawler

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"sync/atomic"
	"time"
)

// Context 爬虫执行上下文
type Context struct {
	visitedUrls         map[string]bool
	Engine              *CrawlEngine
	RequestQueue        chan *Request
	ItemQueue           chan interface{}
	Crawler             *Crawler
	Settings            *Settings
	RequestingCount     int32
	ProcessingItemCount int32
}

// CrawlEngine 爬取引擎
type CrawlEngine struct {
	context    *Context
	crawler    *Crawler
	httpClient *http.Client
}

// Start 启动引擎
func (eng *CrawlEngine) Start() {
	go eng.StartProcessRequests(eng.context)
	go eng.StartProcessItems(eng.context)
}

// StartProcessItems 开始处理Items
func (eng *CrawlEngine) StartProcessItems(ctx *Context) {
	workerCount := int32(6)
	if ctx.Settings.MaxConcurrentProcessItems > 0 {
		workerCount = ctx.Settings.MaxConcurrentProcessItems
	}

	worker := func(ctx *Context) {
		hasInc := false
		defer func() {
			if hasInc {
				atomic.AddInt32(&ctx.ProcessingItemCount, -1)
			}
		}()

		for item := range ctx.ItemQueue {
			atomic.AddInt32(&ctx.ProcessingItemCount, 1)
			hasInc = true

			/* pipeline 执行顺序
			** 1）先执行指定类型的处理函数
			** 2）若上步返回值非空则继续执行通用处理函数
			** 3）若上步返回值非空继续按顺序执行pipeline列表中的pipeline
			 */

			if len(ctx.Crawler.ItemTypeFuncs) > 0 {
				if len(ctx.Crawler.ItemTypeFuncs) > 1 || ctx.Crawler.ItemTypeFuncs["*"] == nil {
					itemType := reflect.TypeOf(item).String()
					if processFunc := ctx.Crawler.ItemTypeFuncs[itemType]; processFunc != nil {
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
			atomic.AddInt32(&ctx.ProcessingItemCount, -1)
			hasInc = false
		}
	}

	for i := int32(0); i < workerCount; i++ {
		go worker(eng.context)
	}
}

// StartProcessRequests 开始处理请求
func (eng *CrawlEngine) StartProcessRequests(ctx *Context) {
	// request = http.Request{Method: "GET"}
	// request.
	fmt.Println(eng, ctx)
	worker := func(req *Request) {
		atomic.AddInt32(&ctx.RequestingCount, 1)
		defer atomic.AddInt32(&ctx.RequestingCount, -1)

		timeout := time.Duration(20 * time.Microsecond)
		if req.Timeout != 0 {
			timeout = time.Duration(req.Timeout) * time.Microsecond
		}
		_context, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		request, err := http.NewRequest(req.Method, req.URL, nil)
		if err != nil {
			fmt.Println(err)
		}
		request.WithContext(_context)

		response, err := eng.httpClient.Do(request)
		if err != nil {
			fmt.Println(err)
		}
		defer response.Body.Close()
		body, _ := ioutil.ReadAll(response.Body)
		// response.Header
		res := NewResponse(body).WithRequest(req).WithStatus(response.StatusCode, response.Status)
		// &Response{Body: body, Status: response.Status, StatusCode: response.StatusCode, Request: req}
		if req.Callback != nil {
			req.Callback(res, ctx)
		} else if eng.crawler.RequestCallback != nil {
			eng.crawler.RequestCallback(res, ctx)
		}
	}
	for req := range ctx.RequestQueue {
		fmt.Println(req)
		// if req.URL == "action::stop" {
		// 	break
		// }
		if ctx.Settings.RequestDelay > 0 {
			worker(req)
			time.Sleep(time.Duration(ctx.Settings.RequestDelay) * time.Millisecond)
		} else {
			for ctx.RequestingCount >= ctx.Settings.MaxConcurrentRequests {
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
	return !eng.context.isProcessingRequests() && !eng.context.isProcessingItems()
}

// AddRequest 添加请求
func (ctx *Context) AddRequest(req *Request) {
	// fmt.Println("add: ", req)
	ctx.RequestQueue <- req
}

// AddItem 处理item
func (ctx *Context) AddItem(item interface{}) {
	// i :=1;
	// fmt.Println(item)
	ctx.ItemQueue <- item
}

// Emit 提交Request或item
func (ctx *Context) Emit(item interface{}) {
	// i :=1;
	switch item.(type) {
	case *Request:
		ctx.AddRequest(item.(*Request))
		break
	case Request:
		request := item.(Request)
		ctx.AddRequest(&request)
		break
	default:
		ctx.AddItem(item)
	}

}

// isProcessingRequests 判断是否有请求正在处理, 或者还未处理
func (ctx *Context) isProcessingRequests() bool {
	return ctx.RequestingCount > 0 || len(ctx.RequestQueue) > 0
}

// isProcessingItems 判断是否有Item正在处理, 或者还未处理
func (ctx *Context) isProcessingItems() bool {
	return ctx.ProcessingItemCount > 0 || len(ctx.ItemQueue) > 0
}
