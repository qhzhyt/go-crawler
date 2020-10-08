package crawler

// import "dcms"

// ItemPipelineFunc 处理item的函数
type ItemPipelineFunc func(item interface{}, ctx *Context) interface{}

// ItemPipeline pipeline接口
type ItemPipeline interface {
	ProcessItem(item interface{}, ctx *Context) interface{}
}

type funcPipeline struct {
	callback ItemPipelineFunc
}

func (cp *funcPipeline) ProcessItem(item interface{}, ctx *Context) interface{} {
	if cp.callback != nil {
		return cp.callback(item, ctx)
	}
	return item
}

// MongoPipeline 默认mongodb pipeline
type MongoPipeline struct {
	MongoDBURI string
	Database   string
	Collection string
}

// ProcessItem 实现ItemPipeline接口
func (dmp *MongoPipeline) ProcessItem(item interface{}, ctx *Context) interface{} {
	// dcms.SaveItem(item)
	return item
}

// DefaultPipeLines 默认的pipelines
func DefaultPipeLines() []ItemPipeline {
	pipeline := MongoPipeline{}
	return []ItemPipeline{&pipeline}
}

// FuncPipeline 仅提供一个函数的pipeline
func FuncPipeline(callback ItemPipelineFunc) ItemPipeline {
	return &funcPipeline{callback}
}
