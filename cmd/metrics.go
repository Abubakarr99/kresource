package cmd

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

// podContainerUsage is one container's instantaneous usage, as reported by
// metrics-server for a single pod.
type podContainerUsage struct {
	cpuMilli int64
	memMi    int64
}

// containerUsage aggregates podContainerUsage across every pod belonging to
// one workload, keyed by container name, so callers can average per replica.
type containerUsage struct {
	cpuMilliSum int64
	memMiSum    int64
	count       int
}

// fetchPodMetrics lists metrics-server's PodMetrics once for the namespace
// so every workload's usage can be looked up in-memory instead of issuing
// one metrics API call per workload.
func fetchPodMetrics(ctx context.Context, metricsClient metricsv.Interface, namespace string) (map[string]map[string]podContainerUsage, error) {
	list, err := metricsClient.MetricsV1beta1().PodMetricses(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	result := make(map[string]map[string]podContainerUsage, len(list.Items))
	for _, pm := range list.Items {
		containers := make(map[string]podContainerUsage, len(pm.Containers))
		for _, c := range pm.Containers {
			containers[c.Name] = podContainerUsage{
				cpuMilli: c.Usage.Cpu().MilliValue(),
				memMi:    c.Usage.Memory().Value() / (1024 * 1024),
			}
		}
		result[pm.Name] = containers
	}
	return result, nil
}

// aggregateUsage sums usage per container name across the pods matching
// selector, so a multi-replica workload gets one averaged figure per
// container instead of one row per pod.
func aggregateUsage(pods []corev1.Pod, podMetrics map[string]map[string]podContainerUsage, selector labels.Selector) map[string]containerUsage {
	result := map[string]containerUsage{}
	if selector == nil || podMetrics == nil {
		return result
	}

	for _, pod := range pods {
		if !selector.Matches(labels.Set(pod.Labels)) {
			continue
		}
		containers, ok := podMetrics[pod.Name]
		if !ok {
			continue
		}
		for name, u := range containers {
			agg := result[name]
			agg.cpuMilliSum += u.cpuMilli
			agg.memMiSum += u.memMi
			agg.count++
			result[name] = agg
		}
	}
	return result
}

// usageForPod builds a single-replica usage map for a standalone pod, which
// has no workload selector to aggregate across.
func usageForPod(podMetrics map[string]map[string]podContainerUsage, podName string) map[string]containerUsage {
	result := map[string]containerUsage{}
	for name, u := range podMetrics[podName] {
		result[name] = containerUsage{cpuMilliSum: u.cpuMilli, memMiSum: u.memMi, count: 1}
	}
	return result
}

func selectorFor(sel *metav1.LabelSelector) labels.Selector {
	if sel == nil {
		return nil
	}
	s, err := metav1.LabelSelectorAsSelector(sel)
	if err != nil {
		return nil
	}
	return s
}
