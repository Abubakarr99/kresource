package cmd

// ResourceRow is one container's resource configuration (and, optionally,
// actual usage averaged across the replicas of its workload).
type ResourceRow struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Container string `json:"container"`
	Init      bool   `json:"init"`

	ReqCPU int64 `json:"requestCpuMilli"`
	ReqMem int64 `json:"requestMemMi"`

	LimCPU    int64 `json:"limitCpuMilli"`
	LimCPUSet bool  `json:"limitCpuSet"`
	LimMem    int64 `json:"limitMemMi"`
	LimMemSet bool  `json:"limitMemSet"`

	// Average actual usage per replica, from metrics-server. Nil when
	// --with-usage wasn't set, metrics-server has no data yet, or the
	// row's kind has no live pods to sample (e.g. CronJob).
	ActualCPU *int64 `json:"actualCpuMilli,omitempty"`
	ActualMem *int64 `json:"actualMemMi,omitempty"`
}

// ResourceTotals aggregates ResourceRow across a whole listing.
type ResourceTotals struct {
	Containers int `json:"containers"`

	ReqCPU int64 `json:"requestCpuMilliTotal"`
	ReqMem int64 `json:"requestMemMiTotal"`

	// LimCPU/LimMem only sum containers that actually set a limit;
	// UnboundedCPUCount/UnboundedMemCount count the rest, since summing
	// "no limit" as 0 would understate how unbounded the namespace is.
	LimCPU            int64 `json:"limitCpuMilliTotal"`
	UnboundedCPUCount int   `json:"unboundedCpuContainers"`
	LimMem            int64 `json:"limitMemMiTotal"`
	UnboundedMemCount int   `json:"unboundedMemContainers"`

	ActualCPU *int64 `json:"actualCpuMilliTotal,omitempty"`
	ActualMem *int64 `json:"actualMemMiTotal,omitempty"`
}

func computeTotals(rows []ResourceRow) ResourceTotals {
	var t ResourceTotals
	var actualCPUSum, actualMemSum int64
	haveActual := false

	for _, r := range rows {
		t.Containers++
		t.ReqCPU += r.ReqCPU
		t.ReqMem += r.ReqMem

		if r.LimCPUSet {
			t.LimCPU += r.LimCPU
		} else {
			t.UnboundedCPUCount++
		}
		if r.LimMemSet {
			t.LimMem += r.LimMem
		} else {
			t.UnboundedMemCount++
		}

		if r.ActualCPU != nil {
			actualCPUSum += *r.ActualCPU
			actualMemSum += *r.ActualMem
			haveActual = true
		}
	}

	if haveActual {
		t.ActualCPU = &actualCPUSum
		t.ActualMem = &actualMemSum
	}
	return t
}
