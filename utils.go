package crawler

import (
	urlLib "net/url"
	"strings"
)

func URLJoin(url string, path string) string {
	if strings.HasPrefix(path, "http") {
		return path
	} else {
		urlObj, _ := urlLib.Parse(url)
		urlPath, _ := urlLib.Parse(path)
		return urlObj.ResolveReference(urlPath).String()
	}
}
