package event

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
)

type (
	// RawData convert []byte from subscription
	RawData []byte
)

func (rd RawData) String() string {
	return string(rd)
}

// Bytes convert raw data to []byte
func (rd RawData) Bytes() []byte {
	return []byte(rd)
}

// Compress raw data
func Compress(v RawData) (RawData, error) {

	var (
		err error
		buf bytes.Buffer
		zpw = gzip.NewWriter(&buf)
	)

	_, err = zpw.Write(v.Bytes())
	if err != nil {
		return nil, err
	}
	err = zpw.Flush()
	if err != nil {
		return nil, err
	}

	return RawData(buf.Bytes()), nil
}

// Uncompress compressed data
func Uncompress(rd RawData) (RawData, error) {

	var (
		err error
		buf = bytes.NewBuffer(rd.Bytes())
		ret bytes.Buffer
	)

	zpr, err := gzip.NewReader(buf)
	if err != nil {
		return nil, err
	}

	io.Copy(&ret, zpr)

	return RawData(ret.Bytes()), nil
}

// Marshal 資料打包
func Marshal(v interface{}) (RawData, error) {
	return json.Marshal(v)
}

// Unmarshal 資料轉換
func Unmarshal(rd RawData, v interface{}) error {
	return json.Unmarshal(rd, v)
}
