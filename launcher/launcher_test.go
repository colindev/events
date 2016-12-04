package launcher

import (
	"errors"
	"testing"

	"github.com/colindev/events/client"
)

func TestForge(t *testing.T) {

	// 僅簡單的檢查一下建構語法
	l := New(client.NewPool(func() (client.Conn, error) {
		return nil, errors.New("test error")
	}, 10))

	if err := l.Fire("", nil); err == nil {
		// 簡單的檢查是否正確回傳errConn
		t.Error("must return errConn")
	} else {
		t.Log(err)
	}

}
