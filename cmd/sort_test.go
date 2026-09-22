package cmd

import "testing"

func namesOf(rows []ResourceRow) []string {
	names := make([]string, len(rows))
	for i, r := range rows {
		names[i] = r.Name
	}
	return names
}

func TestSortRowsEmptyKeyLeavesOrderAlone(t *testing.T) {
	rows := []ResourceRow{{Name: "b"}, {Name: "a"}}
	if err := sortRows(rows, "", false); err != nil {
		t.Fatal(err)
	}
	if got := namesOf(rows); got[0] != "b" || got[1] != "a" {
		t.Fatalf("expected an empty --sort-by to leave order untouched, got %v", got)
	}
}

func TestSortRowsByMemoryLimitAscendingAndReverse(t *testing.T) {
	rows := []ResourceRow{
		{Name: "big", LimMem: 512},
		{Name: "small", LimMem: 8},
		{Name: "mid", LimMem: 64},
	}

	if err := sortRows(rows, "memory-limit", false); err != nil {
		t.Fatal(err)
	}
	if got := namesOf(rows); got[0] != "small" || got[1] != "mid" || got[2] != "big" {
		t.Fatalf("expected ascending memory-limit order, got %v", got)
	}

	if err := sortRows(rows, "memory-limit", true); err != nil {
		t.Fatal(err)
	}
	if got := namesOf(rows); got[0] != "big" || got[1] != "mid" || got[2] != "small" {
		t.Fatalf("expected descending memory-limit order after --reverse, got %v", got)
	}
}

func TestSortRowsStableOnTies(t *testing.T) {
	rows := []ResourceRow{
		{Name: "first", Kind: "Pod"},
		{Name: "second", Kind: "Pod"},
		{Name: "third", Kind: "Pod"},
	}
	if err := sortRows(rows, "kind", false); err != nil {
		t.Fatal(err)
	}
	if got := namesOf(rows); got[0] != "first" || got[1] != "second" || got[2] != "third" {
		t.Fatalf("expected a stable sort to preserve original order on ties, got %v", got)
	}
}

func TestSortRowsActualUsageTreatsNilAsZero(t *testing.T) {
	measured := int64(50)
	rows := []ResourceRow{
		{Name: "has-usage", ActualCPU: &measured},
		{Name: "no-usage", ActualCPU: nil},
	}
	if err := sortRows(rows, "actual-cpu", false); err != nil {
		t.Fatal(err)
	}
	if got := namesOf(rows); got[0] != "no-usage" || got[1] != "has-usage" {
		t.Fatalf("expected nil actual usage to sort as 0 (ascending first), got %v", got)
	}
}

func TestSortRowsUnknownKey(t *testing.T) {
	rows := []ResourceRow{{Name: "a"}}
	if err := sortRows(rows, "bogus", false); err == nil {
		t.Fatal("expected an error for an unknown --sort-by key")
	}
}
