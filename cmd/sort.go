package cmd

import (
	"fmt"
	"sort"
	"strings"
)

const validSortKeysMsg = "kind, name, container, cpu-request, memory-request, cpu-limit, memory-limit, actual-cpu, actual-memory"

// sortRows sorts rows in place by key (case-insensitive), ascending unless
// reverse is set. An empty key leaves rows untouched. Sorting is stable so
// rows with equal keys keep their original (collection) order.
func sortRows(rows []ResourceRow, key string, reverse bool) error {
	if key == "" {
		return nil
	}

	less, err := sortLess(strings.ToLower(strings.TrimSpace(key)))
	if err != nil {
		return err
	}

	sort.SliceStable(rows, func(i, j int) bool {
		if reverse {
			i, j = j, i
		}
		return less(rows[i], rows[j])
	})
	return nil
}

func sortLess(key string) (func(a, b ResourceRow) bool, error) {
	switch key {
	case "kind":
		return func(a, b ResourceRow) bool { return a.Kind < b.Kind }, nil
	case "name":
		return func(a, b ResourceRow) bool { return a.Name < b.Name }, nil
	case "container":
		return func(a, b ResourceRow) bool { return a.Container < b.Container }, nil
	case "cpu-request":
		return func(a, b ResourceRow) bool { return a.ReqCPU < b.ReqCPU }, nil
	case "memory-request":
		return func(a, b ResourceRow) bool { return a.ReqMem < b.ReqMem }, nil
	case "cpu-limit":
		return func(a, b ResourceRow) bool { return a.LimCPU < b.LimCPU }, nil
	case "memory-limit":
		return func(a, b ResourceRow) bool { return a.LimMem < b.LimMem }, nil
	case "actual-cpu":
		return func(a, b ResourceRow) bool { return actualOrZero(a.ActualCPU) < actualOrZero(b.ActualCPU) }, nil
	case "actual-memory":
		return func(a, b ResourceRow) bool { return actualOrZero(a.ActualMem) < actualOrZero(b.ActualMem) }, nil
	default:
		return nil, fmt.Errorf("unknown --sort-by key %q (want one of: %s)", key, validSortKeysMsg)
	}
}

func actualOrZero(v *int64) int64 {
	if v == nil {
		return 0
	}
	return *v
}
