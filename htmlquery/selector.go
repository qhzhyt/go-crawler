package htmlquery

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/antchfx/xpath"
	"github.com/golang/groupcache/lru"
	"golang.org/x/net/html"
)

// Selector Selector
type Selector struct {
	Node   *html.Node
	IsRoot bool
}

// Selectors Selector数组
type Selectors []*Selector

// DisableSelectorCache will disable caching for the query selector if value is true.
var DisableSelectorCache = false

// SelectorCacheMaxEntries allows how many selector object can be caching. Default is 50.
// Will disable caching if SelectorCacheMaxEntries <= 0.
var SelectorCacheMaxEntries = 100

var (
	cacheOnce  sync.Once
	cache      *lru.Cache
	cacheMutex sync.Mutex
)

func getQuery(expr string) (*xpath.Expr, error) {
	if DisableSelectorCache || SelectorCacheMaxEntries <= 0 {
		return xpath.Compile(expr)
	}
	// fmt.Println(1, expr)
	cacheOnce.Do(func() {
		cache = lru.New(SelectorCacheMaxEntries)
	})
	// fmt.Println(2, expr)
	cacheMutex.Lock()
	// fmt.Println(3, expr)
	defer cacheMutex.Unlock()
	if v, ok := cache.Get(expr); ok {
		return v.(*xpath.Expr), nil
	}
	v, err := xpath.Compile(expr)
	if err != nil {
		return nil, err
	}
	cache.Add(expr, v)
	return v, nil
}

func getQueryByCSS(css string, global bool) (*xpath.Expr, error) {
	scope := LOCAL
	if global {
		scope = GLOBAL
	}

	if DisableSelectorCache || SelectorCacheMaxEntries <= 0 {

		return getQuery(CSS2Xpath(css, Scope(scope)))
	}

	cacheOnce.Do(func() {
		cache = lru.New(SelectorCacheMaxEntries)
	})
	cacheMutex.Lock()
	// defer

	if v, ok := cache.Get(css); ok {
		cacheMutex.Unlock()
		return v.(*xpath.Expr), nil
	}
	cacheMutex.Unlock()
	v, err := getQuery(CSS2Xpath(css, Scope(scope)))
	if err != nil {
		return nil, err
	}
	cache.Add(css, v)
	return v, nil
}

// NewSelector 通过bytes 生成selector
func NewSelector(content []byte) *Selector {
	// r, err := charset.(resp.Body, resp.Header.Get("Content-Type"))
	// if err != nil {
	// 	return nil, err
	// }
	node, err := html.Parse(bytes.NewReader(content))
	if err != nil {
		fmt.Println(err)
		return nil
	}
	return &Selector{node, true}
}

// CSS 通过CSS选择节点
func (s *Selector) CSS(css string) Selectors {
	fmt.Println(css)
	xpath, err := getQueryByCSS(css, s.IsRoot)
	fmt.Println("css")
	fmt.Println(CSS2Xpath(css, Scope(LOCAL)))
	if err != nil {
		return Selectors{}
	}
	nodes := QuerySelectorAll(s.Node, xpath)
	selectors := make(Selectors, len(nodes))
	for i, node := range nodes {
		selectors[i] = &Selector{node, false}
	}

	return selectors
}

// Xpath 通过Xpath选择节点
func (s *Selector) Xpath(path string) Selectors {
	xpath, err := getQuery(path)
	if err != nil {
		return Selectors{}
	}
	nodes := QuerySelectorAll(s.Node, xpath)
	selectors := make(Selectors, len(nodes))
	for i, node := range nodes {
		selectors[i] = &Selector{node, false}
	}
	return selectors
}

// Attr 获取节点属性
func (s *Selector) Attr(name string) string {
	for _, attr := range s.Node.Attr {
		if attr.Key == name {
			return attr.Val
		}
	}
	return ""
}

// Text 获取节点内所有文本
func (s *Selector) Text() string {
	var output func(*bytes.Buffer, *html.Node)
	output = func(buf *bytes.Buffer, n *html.Node) {
		switch n.Type {
		case html.TextNode:
			buf.WriteString(n.Data)
			return
		case html.CommentNode:
			return
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			output(buf, child)
		}
	}

	var buf bytes.Buffer
	output(&buf, s.Node)
	return buf.String()
}

// HTML 获取节点完整html代码
func (s *Selector) HTML() string {
	var buf bytes.Buffer
	html.Render(&buf, s.Node)
	return buf.String()
}

// InnerHTML 获取节点内html代码
func (s *Selector) InnerHTML() string {
	var buf bytes.Buffer

	for n := s.Node.FirstChild; n != nil; n = s.Node.NextSibling {
		html.Render(&buf, n)
	}
	return buf.String()
}

// Texts 所有Selector的text列表
func (ss Selectors) Texts() []string {
	texts := make([]string, len(ss))
	for i, s := range ss {
		texts[i] = s.Text()
	}
	return texts
}

// HTMLs 获取节点完整html代码
func (ss Selectors) HTMLs() []string {
	texts := make([]string, len(ss))
	for i, s := range ss {
		texts[i] = s.HTML()
	}
	return texts
}

// InnerHTMLs 获取节点内html代码
func (ss Selectors) InnerHTMLs() []string {
	texts := make([]string, len(ss))
	for i, s := range ss {
		texts[i] = s.InnerHTML()
	}
	return texts
}

// Attrs 获取节点属性
func (ss Selectors) Attrs(name string) []string {
	result := make([]string, len(ss))
	for i, s := range ss {
		result[i] = s.Attr(name)
	}
	return result
}
