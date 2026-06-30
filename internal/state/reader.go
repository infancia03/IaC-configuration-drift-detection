package state

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// Reader loads Terraform state files and extracts raw resources.
type Reader struct{}

func NewReader() *Reader {
	return &Reader{}
}

type tfState struct {
	Version   int          `json:"version"`
	Resources []tfResource `json:"resources"`
}

type tfResource struct {
	Module    string       `json:"module"`
	Mode      string       `json:"mode"`
	Type      string       `json:"type"`
	Name      string       `json:"name"`
	Provider  string       `json:"provider"`
	Instances []tfInstance `json:"instances"`
}

type tfInstance struct {
	Attributes map[string]any `json:"attributes"`
}

// RawResource is a Terraform state resource before normalization.
type RawResource struct {
	Address    string
	Type       string
	Name       string
	Module     string
	Provider   string
	Attributes map[string]any
}

// ReadFile loads resources from a local terraform.tfstate path.
func (r *Reader) ReadFile(_ context.Context, path string) ([]RawResource, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open state file: %w", err)
	}
	defer f.Close()
	return r.Read(f)
}

// Read parses Terraform state JSON from a reader.
func (r *Reader) Read(rd io.Reader) ([]RawResource, error) {
	var state tfState
	if err := json.NewDecoder(rd).Decode(&state); err != nil {
		return nil, fmt.Errorf("decode state: %w", err)
	}
	if state.Version == 0 {
		return nil, fmt.Errorf("invalid or empty terraform state")
	}

	out := make([]RawResource, 0)
	for _, res := range state.Resources {
		if res.Mode != "managed" {
			continue
		}
		address := buildAddress(res.Module, res.Type, res.Name)
		for _, inst := range res.Instances {
			attrs := inst.Attributes
			if attrs == nil {
				attrs = map[string]any{}
			}
			out = append(out, RawResource{
				Address:    address,
				Type:       res.Type,
				Name:       res.Name,
				Module:     res.Module,
				Provider:   res.Provider,
				Attributes: attrs,
			})
		}
	}
	return out, nil
}

func buildAddress(module, typ, name string) string {
	prefix := strings.TrimPrefix(module, "module.")
	if prefix != "" && prefix != module {
		return fmt.Sprintf("module.%s.%s.%s", prefix, typ, name)
	}
	if module != "" {
		return fmt.Sprintf("%s.%s.%s", strings.TrimPrefix(module, "module."), typ, name)
	}
	return fmt.Sprintf("%s.%s", typ, name)
}

// ResourceTypes returns unique Terraform resource types present in state.
func ResourceTypes(resources []RawResource) []string {
	seen := make(map[string]struct{})
	types := make([]string, 0)
	for _, r := range resources {
		if _, ok := seen[r.Type]; ok {
			continue
		}
		seen[r.Type] = struct{}{}
		types = append(types, r.Type)
	}
	return types
}
