package crawler

import (
	"testing"
)

func TestURLJoin(t *testing.T) {
	t.Log(URLJoin("http://www.baidu.com/index/index.php", "login.php"))
	t.Log(URLJoin("http://www.baidu.com/index/index.php", "/login.php"))
}
