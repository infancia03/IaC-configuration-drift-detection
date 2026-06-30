package drift

import (
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/infancia03/IaC-configuration-drift-detection/internal/model"
)

// Options controls comparison behavior.
type Options struct {
	IgnoreAttributes map[string]struct{}
}

// Engine compares expected (state) and actual (cloud) resource sets.
type Engine struct {
	opts Options
}

func NewEngine(opts Options) *Engine {
	return &Engine{opts: opts}
}

// Compare builds a drift report from normalized resource lists.
func (e *Engine) Compare(workspace string, expected, actual []model.Resource) model.DriftReport {
	expectedByID := indexByID(expected)
	actualByID := indexByID(actual)

	report := model.DriftReport{
		ScanID:    uuid.NewString(),
		Timestamp: model.Now(),
		Workspace: workspace,
		Drifts:    make([]model.DriftItem, 0),
	}

	seen := make(map[string]struct{})

	for id, exp := range expectedByID {
		seen[id] = struct{}{}
		act, ok := actualByID[id]
		if !ok {
			report.Drifts = append(report.Drifts, model.DriftItem{
				ResourceID:   id,
				ResourceType: exp.Type,
				ResourceName: exp.Name,
				DriftType:    model.DriftMissingInCloud,
			})
			continue
		}

		attrChanges := diffAttributes(exp.Attributes, act.Attributes, e.opts.IgnoreAttributes)
		tagChanges := diffTags(exp.Tags, act.Tags)

		switch {
		case len(attrChanges) > 0 && len(tagChanges) > 0:
			changes := append(attrChanges, tagChanges...)
			report.Drifts = append(report.Drifts, model.DriftItem{
				ResourceID:   id,
				ResourceType: exp.Type,
				ResourceName: exp.Name,
				DriftType:    model.DriftAttributeChanged,
				Changes:      changes,
			})
		case len(attrChanges) > 0:
			report.Drifts = append(report.Drifts, model.DriftItem{
				ResourceID:   id,
				ResourceType: exp.Type,
				ResourceName: exp.Name,
				DriftType:    model.DriftAttributeChanged,
				Changes:      attrChanges,
			})
		case len(tagChanges) > 0:
			report.Drifts = append(report.Drifts, model.DriftItem{
				ResourceID:   id,
				ResourceType: exp.Type,
				ResourceName: exp.Name,
				DriftType:    model.DriftTagsChanged,
				Changes:      tagChanges,
			})
		}
	}

	for id, act := range actualByID {
		if _, ok := seen[id]; ok {
			continue
		}
		report.Drifts = append(report.Drifts, model.DriftItem{
			ResourceID:   id,
			ResourceType: act.Type,
			ResourceName: act.Name,
			DriftType:    model.DriftExtraInCloud,
		})
	}

	sort.Slice(report.Drifts, func(i, j int) bool {
		if report.Drifts[i].ResourceType != report.Drifts[j].ResourceType {
			return report.Drifts[i].ResourceType < report.Drifts[j].ResourceType
		}
		return report.Drifts[i].ResourceName < report.Drifts[j].ResourceName
	})

	report.Summary = summarize(len(expectedByID), len(actualByID), report.Drifts)
	return report
}

func indexByID(resources []model.Resource) map[string]model.Resource {
	out := make(map[string]model.Resource, len(resources))
	for _, r := range resources {
		out[r.ID] = r
	}
	return out
}

func diffAttributes(expected, actual map[string]any, ignore map[string]struct{}) []model.Change {
	changes := make([]model.Change, 0)
	keys := unionKeys(expected, actual)

	for _, key := range keys {
		if shouldIgnore(key, ignore) {
			continue
		}
		expVal, expOK := expected[key]
		actVal, actOK := actual[key]
		if !expOK && !actOK {
			continue
		}
		if !expOK || !actOK || !valuesEqual(expVal, actVal) {
			changes = append(changes, model.Change{
				Path:     key,
				Expected: expVal,
				Actual:   actVal,
			})
		}
	}
	return changes
}

func diffTags(expected, actual map[string]string) []model.Change {
	exp := normalizeTagMap(expected)
	act := normalizeTagMap(actual)

	changes := make([]model.Change, 0)
	keys := unionStringKeys(exp, act)

	for _, key := range keys {
		expVal, expOK := exp[key]
		actVal, actOK := act[key]
		if expOK == actOK && expVal == actVal {
			continue
		}
		changes = append(changes, model.Change{
			Path:     "tags." + key,
			Expected: expVal,
			Actual:   actVal,
		})
	}
	return changes
}

func normalizeTagMap(tags map[string]string) map[string]string {
	if len(tags) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(tags))
	for k, v := range tags {
		out[strings.ToLower(strings.TrimSpace(k))] = v
	}
	return out
}

func unionKeys(a, b map[string]any) []string {
	seen := make(map[string]struct{})
	for k := range a {
		seen[k] = struct{}{}
	}
	for k := range b {
		seen[k] = struct{}{}
	}
	keys := make([]string, 0, len(seen))
	for k := range seen {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func unionStringKeys(a, b map[string]string) []string {
	seen := make(map[string]struct{})
	for k := range a {
		seen[k] = struct{}{}
	}
	for k := range b {
		seen[k] = struct{}{}
	}
	keys := make([]string, 0, len(seen))
	for k := range seen {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func shouldIgnore(path string, ignore map[string]struct{}) bool {
	if len(ignore) == 0 {
		return false
	}
	if _, ok := ignore[path]; ok {
		return true
	}
	for pattern := range ignore {
		if strings.HasSuffix(pattern, "*") {
			prefix := strings.TrimSuffix(pattern, "*")
			if strings.HasPrefix(path, prefix) {
				return true
			}
		}
	}
	return false
}

func valuesEqual(a, b any) bool {
	switch av := a.(type) {
	case string:
		bv, ok := b.(string)
		return ok && av == bv
	case bool:
		bv, ok := b.(bool)
		return ok && av == bv
	case float64:
		bv, ok := b.(float64)
		return ok && av == bv
	case int:
		bv, ok := b.(int)
		return ok && av == bv
	case int64:
		bv, ok := b.(int64)
		return ok && av == bv
	case []any:
		bv, ok := b.([]any)
		if !ok {
			return false
		}
		if len(av) != len(bv) {
			return false
		}
		for i := range av {
			if !valuesEqual(av[i], bv[i]) {
				return false
			}
		}
		return true
	case map[string]any:
		bv, ok := b.(map[string]any)
		if !ok {
			return false
		}
		if len(av) != len(bv) {
			return false
		}
		for k, v := range av {
			if !valuesEqual(v, bv[k]) {
				return false
			}
		}
		return true
	default:
		return a == b
	}
}

func summarize(expectedCount, actualCount int, drifts []model.DriftItem) model.Summary {
	summary := model.Summary{
		TotalExpected: expectedCount,
		TotalActual:   actualCount,
	}

	for _, d := range drifts {
		switch d.DriftType {
		case model.DriftMissingInCloud:
			summary.Missing++
		case model.DriftExtraInCloud:
			summary.Extra++
		case model.DriftAttributeChanged:
			summary.Modified++
		case model.DriftTagsChanged:
			summary.TagsChanged++
		}
	}

	drifted := summary.Missing + summary.Extra + summary.Modified + summary.TagsChanged
	if expectedCount >= drifted {
		summary.Unchanged = expectedCount - summary.Missing - summary.Modified - summary.TagsChanged
	}
	return summary
}
