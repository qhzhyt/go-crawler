package htmlquery

import "testing"

func TestGetQuery(t *testing.T) {
	a, nil := getQueryByCSS("a", true)
	t.Log(a, nil)
}
