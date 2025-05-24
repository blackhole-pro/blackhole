package state

import (
	"encoding/json"
)

// JSONStateSerializer implements StateSerializer using JSON encoding
type JSONStateSerializer struct{}

// NewJSONStateSerializer creates a new JSON state serializer
func NewJSONStateSerializer() StateSerializer {
	return &JSONStateSerializer{}
}

// Serialize converts plugin state to JSON bytes
func (s *JSONStateSerializer) Serialize(state interface{}) ([]byte, error) {
	return json.Marshal(state)
}

// Deserialize converts JSON bytes to plugin state
func (s *JSONStateSerializer) Deserialize(data []byte, target interface{}) error {
	return json.Unmarshal(data, target)
}