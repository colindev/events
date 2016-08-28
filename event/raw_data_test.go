package event

import "testing"

func TestMarshal(t *testing.T) {
	var (
		rd  RawData
		err error
	)

	data := map[string]interface{}{
		"A":         "A",
		"abc":       "abc",
		`{"A":123}`: struct{ A int }{123},
	}

	for s, o := range data {
		rd, err = Marshal(o)
		if err != nil || rd.String() != s {
			t.Errorf("Marshal('a') return %v, %v\n", rd, err)
		}
	}

	if RawData("A").String() != "A" {
		t.Error("RawData convert string fail")
	}
}
