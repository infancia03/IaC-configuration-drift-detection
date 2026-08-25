package report

import (
	"encoding/json"
	"fmt"
	"io"
)

// WriteJSON writes a value as indented JSON.
func WriteJSON(w io.Writer, value any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(value); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}
	return nil
}
