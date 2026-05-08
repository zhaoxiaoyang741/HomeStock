package taskcenter

import "encoding/json"

type changes struct {
	Before any `json:"before,omitempty"`
	After  any `json:"after,omitempty"`
}

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
