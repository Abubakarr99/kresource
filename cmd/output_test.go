package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func sampleRowsAndTotals() ([]ResourceRow, ResourceTotals) {
	unbounded := int64(0)
	rows := []ResourceRow{
		{
			Kind: "Deployment", Name: "web", Container: "app",
			ReqCPU: 100, ReqMem: 128, LimCPU: 200, LimCPUSet: true,
			ActualCPU: &unbounded, ActualMem: &unbounded,
		},
		{
			Kind: "Pod", Name: "standalone", Container: "bare",
			ReqCPU: 50,
		},
	}
	return rows, computeTotals(rows)
}

func TestComputeTotalsUnboundedCounting(t *testing.T) {
	rows, totals := sampleRowsAndTotals()
	if totals.Containers != len(rows) {
		t.Fatalf("expected %d containers, got %d", len(rows), totals.Containers)
	}
	if totals.ReqCPU != 150 {
		t.Fatalf("expected total request cpu 150m, got %d", totals.ReqCPU)
	}
	// Only the Deployment row set a CPU limit; both rows left memory unset.
	if totals.UnboundedCPUCount != 1 || totals.UnboundedMemCount != 2 {
		t.Fatalf("unexpected unbounded counts: %+v", totals)
	}
	if totals.LimCPU != 200 {
		t.Fatalf("expected summed cpu limit 200m (ignoring the unset one), got %d", totals.LimCPU)
	}
}

func TestRenderTableShowsNoneForUnsetLimits(t *testing.T) {
	rows, totals := sampleRowsAndTotals()
	var buf bytes.Buffer
	renderTable(&buf, rows, totals, true)
	out := buf.String()

	if !strings.Contains(out, "none") {
		t.Fatalf("expected unset limits to render as 'none', got:\n%s", out)
	}
	// tablewriter uppercases footer text by default, hence the case-insensitive check.
	if !strings.Contains(strings.ToLower(out), "unbounded") {
		t.Fatalf("expected totals footer to call out unbounded containers, got:\n%s", out)
	}
	if !strings.Contains(out, "n/a") {
		t.Fatalf("expected the bare pod row (no usage data) to render 'n/a', got:\n%s", out)
	}
}

func TestRenderJSONRoundTrips(t *testing.T) {
	rows, totals := sampleRowsAndTotals()
	var buf bytes.Buffer
	if err := renderJSON(&buf, rows, totals); err != nil {
		t.Fatal(err)
	}

	var out struct {
		Rows   []ResourceRow  `json:"rows"`
		Totals ResourceTotals `json:"totals"`
	}
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("output wasn't valid JSON: %v\n%s", err, buf.String())
	}
	if len(out.Rows) != len(rows) || out.Totals.Containers != totals.Containers {
		t.Fatalf("round-tripped JSON doesn't match input: %+v", out)
	}
	if strings.Contains(buf.String(), `"actualCpuMilli"`) == false {
		t.Fatalf("expected actualCpuMilli present when set, got:\n%s", buf.String())
	}
}

func TestRenderCSVOmitsTotalsAndHandlesMissingUsage(t *testing.T) {
	rows, _ := sampleRowsAndTotals()
	var buf bytes.Buffer
	if err := renderCSV(&buf, rows, true); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != len(rows)+1 { // header + one line per row, no totals line
		t.Fatalf("expected %d lines (header+rows), got %d:\n%s", len(rows)+1, len(lines), buf.String())
	}
	if !strings.Contains(lines[0], "actualCpuMilli") {
		t.Fatalf("expected usage columns in header, got %q", lines[0])
	}
	// The bare pod row has no usage data; its actual-usage fields must be blank, not "0".
	if !strings.HasSuffix(lines[2], ",,") {
		t.Fatalf("expected trailing empty actual-usage fields for the bare pod row, got %q", lines[2])
	}
}

func TestRenderYAMLValid(t *testing.T) {
	rows, totals := sampleRowsAndTotals()
	var buf bytes.Buffer
	if err := renderYAML(&buf, rows, totals); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "kind: Deployment") {
		t.Fatalf("expected yaml to contain row data, got:\n%s", buf.String())
	}
}
