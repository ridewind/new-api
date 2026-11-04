package dto

import "encoding/json"

func unmarshalForTest(s string, v any) error { return json.Unmarshal([]byte(s), v) }
