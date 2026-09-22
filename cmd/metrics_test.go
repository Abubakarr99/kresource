package cmd

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	metricsapi "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	metricsfake "k8s.io/metrics/pkg/client/clientset/versioned/fake"
)

// podMetricsGVR is the metrics.k8s.io API's actual REST resource name.
// NewSimpleClientset(objects...) can't register PodMetrics objects
// correctly by itself: its object tracker infers the resource name by
// naively pluralizing the Kind ("PodMetrics" -> "podmetricses"), but the
// real API (and the fake typed client's List/Get calls) use "pods". Adding
// through the tracker with the explicit GVR sidesteps that mismatch.
var podMetricsGVR = schema.GroupVersionResource{Group: "metrics.k8s.io", Version: "v1beta1", Resource: "pods"}

func TestFetchPodMetrics(t *testing.T) {
	ctx := context.Background()
	mc := metricsfake.NewSimpleClientset()
	if err := mc.Tracker().Create(podMetricsGVR, &metricsapi.PodMetrics{
		ObjectMeta: metav1.ObjectMeta{Name: "web-1", Namespace: "ns"},
		Containers: []metricsapi.ContainerMetrics{{
			Name: "app",
			Usage: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("40m"),
				corev1.ResourceMemory: resource.MustParse("64Mi"),
			},
		}},
	}, "ns"); err != nil {
		t.Fatal(err)
	}

	podMetrics, err := fetchPodMetrics(ctx, mc, "ns")
	if err != nil {
		t.Fatal(err)
	}

	usage, ok := podMetrics["web-1"]["app"]
	if !ok {
		t.Fatalf("expected usage for web-1/app, got %+v", podMetrics)
	}
	if usage.cpuMilli != 40 || usage.memMi != 64 {
		t.Fatalf("expected 40m/64Mi, got %+v", usage)
	}
}

func TestAggregateUsageAveragesAcrossReplicas(t *testing.T) {
	pods := []corev1.Pod{
		{ObjectMeta: metav1.ObjectMeta{Name: "web-1", Labels: map[string]string{"app": "web"}}},
		{ObjectMeta: metav1.ObjectMeta{Name: "web-2", Labels: map[string]string{"app": "web"}}},
		{ObjectMeta: metav1.ObjectMeta{Name: "other-1", Labels: map[string]string{"app": "other"}}},
	}
	podMetrics := map[string]map[string]podContainerUsage{
		"web-1":   {"app": {cpuMilli: 40, memMi: 64}},
		"web-2":   {"app": {cpuMilli: 60, memMi: 96}},
		"other-1": {"app": {cpuMilli: 999, memMi: 999}},
	}

	sel := labels.SelectorFromSet(labels.Set{"app": "web"})
	usage := aggregateUsage(pods, podMetrics, sel)

	app, ok := usage["app"]
	if !ok || app.count != 2 {
		t.Fatalf("expected 2 matched replicas for app, got %+v", usage)
	}
	if app.cpuMilliSum != 100 || app.memMiSum != 160 {
		t.Fatalf("expected summed usage 100m/160Mi across replicas, got %+v", app)
	}
}

func TestAggregateUsageNilSelectorOrMetrics(t *testing.T) {
	pods := []corev1.Pod{{ObjectMeta: metav1.ObjectMeta{Name: "web-1", Labels: map[string]string{"app": "web"}}}}
	podMetrics := map[string]map[string]podContainerUsage{"web-1": {"app": {cpuMilli: 40}}}

	if usage := aggregateUsage(pods, podMetrics, nil); len(usage) != 0 {
		t.Fatalf("expected no usage with nil selector, got %+v", usage)
	}
	sel := labels.SelectorFromSet(labels.Set{"app": "web"})
	if usage := aggregateUsage(pods, nil, sel); len(usage) != 0 {
		t.Fatalf("expected no usage with nil metrics, got %+v", usage)
	}
}

func TestUsageForPod(t *testing.T) {
	podMetrics := map[string]map[string]podContainerUsage{
		"bare-pod": {"app": {cpuMilli: 40, memMi: 28}},
	}
	usage := usageForPod(podMetrics, "bare-pod")
	if usage["app"].count != 1 || usage["app"].cpuMilliSum != 40 {
		t.Fatalf("unexpected usage: %+v", usage)
	}
	if got := usageForPod(podMetrics, "missing"); len(got) != 0 {
		t.Fatalf("expected empty map for unknown pod, got %+v", got)
	}
}
