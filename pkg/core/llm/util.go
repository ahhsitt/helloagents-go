package llm

import "encoding/json"

// marshalJSON JSON 序列化
func marshalJSON(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// unmarshalJSON JSON 反序列化
func unmarshalJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
