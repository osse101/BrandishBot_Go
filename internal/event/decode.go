package event

import "encoding/json"

// DecodePayload decodes an event payload into T via type assertion then JSON fallback.
// When events are published via in-process MemoryBus, the payload is already the correct struct.
// When coming from serialized sources, the fallback JSON round-trip handles the conversion.
func DecodePayload[T any](input interface{}) (T, error) {
	if v, ok := input.(T); ok {
		return v, nil
	}
	var result T
	data, err := json.Marshal(input)
	if err != nil {
		return result, err
	}
	return result, json.Unmarshal(data, &result)
}
