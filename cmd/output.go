package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"github.com/olekukonko/tablewriter"
	"sigs.k8s.io/yaml"
)

func renderTable(w io.Writer, rows []ResourceRow, totals ResourceTotals, withUsage bool) {
	table := tablewriter.NewWriter(w)
	header := []string{"Kind", "Name", "Container", "Req CPU", "Req Mem", "Lim CPU", "Lim Mem"}
	if withUsage {
		header = append(header, "Actual CPU", "Actual Mem")
	}
	table.SetHeader(header)

	for _, r := range rows {
		container := r.Container
		if r.Init {
			container += " (init)"
		}
		row := []string{
			r.Kind, r.Name, container,
			fmt.Sprintf("%dm", r.ReqCPU),
			fmt.Sprintf("%dMi", r.ReqMem),
			limitString(r.LimCPU, r.LimCPUSet, "m"),
			limitString(r.LimMem, r.LimMemSet, "Mi"),
		}
		if withUsage {
			row = append(row, actualString(r.ActualCPU, "m"), actualString(r.ActualMem, "Mi"))
		}
		table.Append(row)
	}

	footer := []string{"TOTAL", "", fmt.Sprintf("%d containers", totals.Containers),
		fmt.Sprintf("%dm", totals.ReqCPU),
		fmt.Sprintf("%dMi", totals.ReqMem),
		totalLimitString(totals.LimCPU, totals.UnboundedCPUCount, "m"),
		totalLimitString(totals.LimMem, totals.UnboundedMemCount, "Mi"),
	}
	if withUsage {
		footer = append(footer, actualString(totals.ActualCPU, "m"), actualString(totals.ActualMem, "Mi"))
	}
	table.SetFooter(footer)
	table.Render()
}

func limitString(v int64, set bool, unit string) string {
	if !set {
		return "none"
	}
	return fmt.Sprintf("%d%s", v, unit)
}

func totalLimitString(sum int64, unboundedCount int, unit string) string {
	s := fmt.Sprintf("%d%s", sum, unit)
	if unboundedCount > 0 {
		s += fmt.Sprintf(" (+%d unbounded)", unboundedCount)
	}
	return s
}

func actualString(v *int64, unit string) string {
	if v == nil {
		return "n/a"
	}
	return fmt.Sprintf("%d%s", *v, unit)
}

func renderJSON(w io.Writer, rows []ResourceRow, totals ResourceTotals) error {
	out := struct {
		Rows   []ResourceRow  `json:"rows"`
		Totals ResourceTotals `json:"totals"`
	}{rows, totals}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func renderYAML(w io.Writer, rows []ResourceRow, totals ResourceTotals) error {
	out := struct {
		Rows   []ResourceRow  `json:"rows"`
		Totals ResourceTotals `json:"totals"`
	}{rows, totals}
	b, err := yaml.Marshal(out)
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}

// renderCSV intentionally emits only the raw rows, one schema per line —
// totals are a different shape and CSV consumers (spreadsheets, pandas)
// already compute sums trivially from the raw data.
func renderCSV(w io.Writer, rows []ResourceRow, withUsage bool) error {
	cw := csv.NewWriter(w)
	defer cw.Flush()

	header := []string{"kind", "name", "container", "init", "requestCpuMilli", "requestMemMi", "limitCpuMilli", "limitCpuSet", "limitMemMi", "limitMemSet"}
	if withUsage {
		header = append(header, "actualCpuMilli", "actualMemMi")
	}
	if err := cw.Write(header); err != nil {
		return err
	}

	for _, r := range rows {
		record := []string{
			r.Kind, r.Name, r.Container, strconv.FormatBool(r.Init),
			strconv.FormatInt(r.ReqCPU, 10), strconv.FormatInt(r.ReqMem, 10),
			strconv.FormatInt(r.LimCPU, 10), strconv.FormatBool(r.LimCPUSet),
			strconv.FormatInt(r.LimMem, 10), strconv.FormatBool(r.LimMemSet),
		}
		if withUsage {
			record = append(record, optionalInt(r.ActualCPU), optionalInt(r.ActualMem))
		}
		if err := cw.Write(record); err != nil {
			return err
		}
	}
	return cw.Error()
}

func optionalInt(v *int64) string {
	if v == nil {
		return ""
	}
	return strconv.FormatInt(*v, 10)
}
