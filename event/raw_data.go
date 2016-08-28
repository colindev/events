package event

import "encoding/json"

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

// Marshal value to RawData
func Marshal(v interface{}) (RawData, error) {
	switch v := v.(type) {
	case string:
		return RawData([]byte(v)), nil
	case []byte:
		return RawData(v), nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	return RawData(b), nil
}

// Unmarshal RawData to value
func Unmarshal(rd RawData, v interface{}) error {
	return json.Unmarshal(rd.Bytes(), v)
}
