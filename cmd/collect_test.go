package cmd

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func mustReq(cpu, mem string) corev1.ResourceRequirements {
	return corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse(cpu),
			corev1.ResourceMemory: resource.MustParse(mem),
		},
	}
}

func podSpecWithContainer(name string, resources corev1.ResourceRequirements) corev1.PodSpec {
	return corev1.PodSpec{Containers: []corev1.Container{{Name: name, Resources: resources}}}
}

func TestCollectStatefulSets(t *testing.T) {
	ctx := context.Background()
	cs := fake.NewSimpleClientset(&appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: "db", Namespace: "ns"},
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "db"}},
			Template: corev1.PodTemplateSpec{
				Spec: podSpecWithContainer("app", mustReq("25m", "20Mi")),
			},
		},
	})

	rows, err := collectStatefulSets(ctx, cs, "ns", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Kind != "StatefulSet" || rows[0].Name != "db" || rows[0].ReqCPU != 25 || rows[0].ReqMem != 20 {
		t.Fatalf("unexpected rows: %+v", rows)
	}
}

func TestCollectDaemonSets(t *testing.T) {
	ctx := context.Background()
	cs := fake.NewSimpleClientset(&appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{Name: "agent", Namespace: "ns"},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "agent"}},
			Template: corev1.PodTemplateSpec{
				Spec: podSpecWithContainer("app", mustReq("5m", "8Mi")),
			},
		},
	})

	rows, err := collectDaemonSets(ctx, cs, "ns", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Kind != "DaemonSet" || rows[0].Name != "agent" {
		t.Fatalf("unexpected rows: %+v", rows)
	}
}

func TestCollectReplicaSetsSkipsDeploymentOwned(t *testing.T) {
	ctx := context.Background()
	truthy := true
	cs := fake.NewSimpleClientset(
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Name: "web-abc123", Namespace: "ns",
				OwnerReferences: []metav1.OwnerReference{{Kind: "Deployment", Name: "web", Controller: &truthy}},
			},
			Spec: appsv1.ReplicaSetSpec{
				Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "web"}},
				Template: corev1.PodTemplateSpec{Spec: podSpecWithContainer("app", mustReq("50m", "32Mi"))},
			},
		},
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{Name: "standalone-rs", Namespace: "ns"},
			Spec: appsv1.ReplicaSetSpec{
				Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "standalone-rs"}},
				Template: corev1.PodTemplateSpec{Spec: podSpecWithContainer("app", mustReq("15m", "12Mi"))},
			},
		},
	)

	rows, err := collectReplicaSets(ctx, cs, "ns", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected only the standalone ReplicaSet, got %d rows: %+v", len(rows), rows)
	}
	if rows[0].Name != "standalone-rs" {
		t.Fatalf("expected standalone-rs, got %+v", rows[0])
	}
}

func TestCollectJobsSkipsCronJobOwned(t *testing.T) {
	ctx := context.Background()
	truthy := true
	cs := fake.NewSimpleClientset(
		&batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name: "nightly-28871234", Namespace: "ns",
				OwnerReferences: []metav1.OwnerReference{{Kind: "CronJob", Name: "nightly", Controller: &truthy}},
			},
			Spec: batchv1.JobSpec{
				Template: corev1.PodTemplateSpec{Spec: podSpecWithContainer("app", mustReq("5m", "8Mi"))},
			},
		},
		&batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{Name: "standalone-job", Namespace: "ns"},
			Spec: batchv1.JobSpec{
				Template: corev1.PodTemplateSpec{Spec: podSpecWithContainer("app", mustReq("30m", "24Mi"))},
			},
		},
	)

	rows, err := collectJobs(ctx, cs, "ns", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Name != "standalone-job" {
		t.Fatalf("expected only the standalone Job, got %+v", rows)
	}
}

func TestCollectCronJobsUsesJobTemplate(t *testing.T) {
	ctx := context.Background()
	cs := fake.NewSimpleClientset(&batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{Name: "nightly", Namespace: "ns"},
		Spec: batchv1.CronJobSpec{
			Schedule: "0 0 30 2 *",
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{Spec: podSpecWithContainer("app", mustReq("5m", "8Mi"))},
				},
			},
		},
	})

	rows, err := collectCronJobs(ctx, cs, "ns")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Kind != "CronJob" || rows[0].Name != "nightly" {
		t.Fatalf("unexpected rows: %+v", rows)
	}
	if rows[0].ActualCPU != nil {
		t.Fatalf("CronJob rows should never carry usage data, got %+v", rows[0])
	}
}

func TestCollectBarePodsSkipsOwned(t *testing.T) {
	truthy := true
	pods := []corev1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "web-1", Namespace: "ns",
				OwnerReferences: []metav1.OwnerReference{{Kind: "ReplicaSet", Name: "web-abc123", Controller: &truthy}},
			},
			Spec: podSpecWithContainer("app", corev1.ResourceRequirements{}),
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "bare-pod", Namespace: "ns"},
			Spec:       podSpecWithContainer("app", mustReq("40m", "28Mi")),
		},
	}

	rows := collectBarePods(pods, nil)
	if len(rows) != 1 || rows[0].Name != "bare-pod" || rows[0].Kind != "Pod" {
		t.Fatalf("expected only the unowned pod, got %+v", rows)
	}
}

func TestRowsFromContainersUnsetLimits(t *testing.T) {
	rows := rowsFromContainers("Pod", "p", []corev1.Container{{
		Name: "app",
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("40m")},
			// No Limits set at all.
		},
	}}, false, nil)

	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	r := rows[0]
	if r.LimCPUSet || r.LimMemSet {
		t.Fatalf("expected limits to be reported as unset, got %+v", r)
	}
	if r.ReqMem != 0 {
		t.Fatalf("expected requestless memory to be 0, got %d", r.ReqMem)
	}
}
