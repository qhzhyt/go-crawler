package crawler

// Context 爬虫执行上下文
type Context struct {
	Engine       *CrawlEngine
	Crawler      *Crawler
	Settings     *Settings
	Depth        int32
	LastRequest  *Request
	LastResponse *Response
}

func (ctx *Context) copy() *Context {
	return &Context{
		Engine:       ctx.Engine,
		Crawler:      ctx.Crawler,
		Settings:     ctx.Settings,
		Depth:        ctx.Depth,
		LastRequest:  ctx.LastRequest,
		LastResponse: ctx.LastResponse,
	}
}

// AddRequest 添加请求
func (ctx *Context) AddRequest(req *Request) {
	// fmt.Println("add: ", req)
	if req == nil {
		return
	}
	context := ctx.copy()
	context.Depth++
	if context.LastResponse != nil && req.Headers.Get("Referer") == "" && req.Headers.Get("referer") == "" {
		req.Headers.Set("Referer", context.LastResponse.URL)
	}
	context.LastRequest = req
	req.context = context
	ctx.Engine.RequestQueue <- req
}

// AddItem 处理item
func (ctx *Context) AddItem(item interface{}) {
	// i :=1;
	// fmt.Println(item)
	ctx.Engine.ItemQueue <- &itemWrapper{item: item, context: ctx}
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

func (ctx *Context) retry(req *Request) {
	ctx.Engine.RequestQueue <- req
}
