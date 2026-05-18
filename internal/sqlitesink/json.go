package sqlitesink

import "encoding/json"

// jsonMarshal is a tiny indirection so the test file can override it if
// needed. In practice it's just encoding/json.
func jsonMarshal(v any) ([]byte, error) { return json.Marshal(v) }
