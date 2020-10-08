package crawler

import (
	"net/http"
	"reflect"
)

// type CrawlerCallbacks{
// 	OnStart
// }

// type CrawlerCallback func()

// Crawler 爬虫本体
type Crawler struct {
	Name          string
	StartUrls     []string
	StartRequests func(ctx *Context) []*Request

	onStop          func(ctx *Context)
	onStart         func(ctx *Context)
	Pipelines       []ItemPipeline
	ItemTypeFuncs   map[string]ItemPipelineFunc
	RequestCallback func(res *Response, ctx *Context)
	Settings        *Settings
	Engine          *CrawlEngine
	context         *Context
}

// NewCrawler 创建一个爬虫
func NewCrawler(name string) *Crawler {
	settings := DefaultSettings()
	context := &Context{Settings: settings, RequestQueue: make(chan *Request, 100), ItemQueue: make(chan interface{}, 100)}
	engine := &CrawlEngine{context: context, httpClient: &http.Client{}}
	context.Engine = engine

	crawler := &Crawler{
		Name:          name,
		context:       context,
		Settings:      settings,
		Pipelines:     DefaultPipeLines(),
		ItemTypeFuncs: make(map[string]ItemPipelineFunc),
		Engine:        engine,
	}

	engine.crawler = crawler
	context.Crawler = crawler
	return crawler
}

// startRequests 默认StartRequests
func (c *Crawler) startRequests(ctx *Context) []*Request {
	if c.StartRequests != nil {
		return c.StartRequests(ctx)
	} else if c.StartUrls != nil && len(c.StartUrls) > 0 {
		for _, url := range c.StartUrls {
			c.context.Emit(GetURL(url))
		}
	}
	return []*Request{}
}

// Wait 等待引擎进入空闲状态
func (c *Crawler) Wait() {
	c.Engine.Wait()
}

// Start 启动爬虫
func (c *Crawler) Start(wait bool) *Crawler {

	worker := func() {
		for _, req := range c.startRequests(c.context) {
			c.context.Emit(req)
		}
	}
	c.Engine.Start()
	if wait {
		worker()
		c.Wait()
	} else {
		go worker()
	}
	return c
}

// OnStart 设置start回调
func (c *Crawler) OnStart(callback func(ctx *Context)) *Crawler {
	c.onStart = callback
	return c
}

// OnStop 设置stop回调
func (c *Crawler) OnStop(callback func(ctx *Context)) *Crawler {
	c.onStop = callback
	return c
}

// WithSettings 设置settings
func (c *Crawler) WithSettings(s *Settings) *Crawler {
	if s.MaxConcurrentProcessItems > 0 {
		c.Settings.MaxConcurrentProcessItems = s.MaxConcurrentProcessItems
	}
	if s.MaxConcurrentRequests > 0 {
		c.Settings.MaxConcurrentRequests = s.MaxConcurrentRequests
	}
	if s.RequestDelay > 0 {
		c.Settings.RequestDelay = s.RequestDelay
	}
	if s.RequestTimeout > 0 {
		c.Settings.RequestTimeout = s.RequestTimeout
	}
	return c
}

// AddItemPipeline 添加Pipeline
func (c *Crawler) AddItemPipeline(p ItemPipeline) *Crawler {
	c.Pipelines = append(c.Pipelines, p)
	return c
}

// AddItemPipelineFunc 添加Pipeline func
func (c *Crawler) AddItemPipelineFunc(f ItemPipelineFunc) *Crawler {
	c.Pipelines = append(c.Pipelines, FuncPipeline(f))
	return c
}

// ClearPipelines 清空pipeline
func (c *Crawler) ClearPipelines() *Crawler {
	c.Pipelines = []ItemPipeline{}
	return c
}

// OnItem 默认item处理函数
func (c *Crawler) OnItem(f ItemPipelineFunc) *Crawler {
	// ta :=
	c.ItemTypeFuncs["*"] = f
	return c
}

// OnItemType 与itemExample同类型的item处理函数
func (c *Crawler) OnItemType(itemExample interface{}, f ItemPipelineFunc) *Crawler {
	c.ItemTypeFuncs[reflect.TypeOf(itemExample).String()] = f
	return c
}

// WithDefaultCallback 设置默认回调函数
func (c *Crawler) WithDefaultCallback(callback func(res *Response, ctx *Context)) *Crawler {
	c.RequestCallback = callback
	return c
}

// WithStartRequest 自定义request
func (c *Crawler) WithStartRequest(callback func(ctx *Context) []*Request) *Crawler {
	c.StartRequests = callback
	return c
}
