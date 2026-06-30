package report

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/infancia03/IaC-configuration-drift-detection/internal/model"
)

// WriteJSON writes a drift report as indented JSON.
func WriteJSON(w io.Writer, report model.DriftReport) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		return fmt.Errorf("encode report: %w", err)
	}
	return nil
}
