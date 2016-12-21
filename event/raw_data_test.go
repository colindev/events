package event

import (
	"bytes"
	"testing"
)

func TestCompress(t *testing.T) {
	var (
		err error
		raw = RawData("xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx")
	)

	t.Logf("raw: [%s]", raw)
	s, err := Compress(raw)
	if err != nil {
		t.Error("compress error:", err)
	}

	t.Logf("compressed: [%s]", s)
	if len(raw.Bytes()) <= len(s.Bytes()) {
		t.Error("compressed must smaller then raw")
	}

	back, err := Uncompress(s)
	t.Logf("uncompressed: [%s]", back)
	if !bytes.Equal(back.Bytes(), raw.Bytes()) {
		t.Error("uncompress fail: different data")
	}
}
