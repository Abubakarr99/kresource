package cmd

import (
	"fmt"
	"strconv"
	"strings"
)

// rowFilter narrows rows to those matching every constraint set on it.
// A zero-value rowFilter matches everything.
type rowFilter struct {
	kind      string // matched case-insensitively, exact, against Kind
	name      string // matched case-insensitively, substring, against Name
	container string // matched case-insensitively, substring, against Container
	init      *bool  // nil = no constraint on Init
}

// parseFilter parses a comma-separated key=value list (e.g.
// "type=deployment,name=web") into a rowFilter. Every key must match; an
// empty expr matches everything.
func parseFilter(expr string) (rowFilter, error) {
	var f rowFilter
	if strings.TrimSpace(expr) == "" {
		return f, nil
	}

	for _, pair := range strings.Split(expr, ",") {
		key, value, ok := strings.Cut(pair, "=")
		if !ok {
			return rowFilter{}, fmt.Errorf("invalid --filter %q: expected key=value", pair)
		}
		key = strings.ToLower(strings.TrimSpace(key))
		value = strings.TrimSpace(value)

		switch key {
		case "type", "kind":
			f.kind = value
		case "name":
			f.name = value
		case "container":
			f.container = value
		case "init":
			b, err := strconv.ParseBool(value)
			if err != nil {
				return rowFilter{}, fmt.Errorf("invalid --filter init=%q: want true or false", value)
			}
			f.init = &b
		default:
			return rowFilter{}, fmt.Errorf("unknown --filter key %q (want one of: type, name, container, init)", key)
		}
	}
	return f, nil
}

func (f rowFilter) matches(r ResourceRow) bool {
	if f.kind != "" && !strings.EqualFold(f.kind, r.Kind) {
		return false
	}
	if f.name != "" && !strings.Contains(strings.ToLower(r.Name), strings.ToLower(f.name)) {
		return false
	}
	if f.container != "" && !strings.Contains(strings.ToLower(r.Container), strings.ToLower(f.container)) {
		return false
	}
	if f.init != nil && *f.init != r.Init {
		return false
	}
	return true
}

func filterRows(rows []ResourceRow, f rowFilter) []ResourceRow {
	out := make([]ResourceRow, 0, len(rows))
	for _, r := range rows {
		if f.matches(r) {
			out = append(out, r)
		}
	}
	return out
}
