## go-crawler

A simple crawler library implement with Golang, some features same as Scrapy(a python crawler framework)

## usage

``` golang
import "github.com/qhzhyt/go-crawler"

func parse(res *crawler.Response, ctx *crawler.Context) {
	title := res.CSS("a").Attrs("href")
	ctx.Emit(map[string]interface{}{"title": title})
}

func StartRequests(ctx *crawler.Context) []*crawler.Request {
	return crawler.GetURLS(
        "http://www.baidu.com/",
        "http://www.qq.com/"
    )
}

func main() {
    crawler.NewCrawler("test").
		WithStartRequest(startRequest).
		WithDefaultCallback(parse).
		OnItem(func(item interface{}, ctx *crawler.Context) interface{} {
				fmt.Println(item)
				return nil
		}).
		WithSettings(&crawler.Settings{RequestDelay: 1000}).
		Start(true)
}


```