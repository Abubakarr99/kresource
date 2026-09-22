package cmd

import (
	"bytes"
	"context"
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestRunListEndToEnd(t *testing.T) {
	ctx := context.Background()
	truthy := true
	cs := fake.NewSimpleClientset(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "ns"},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "web"}},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "web"}},
					Spec:       podSpecWithContainer("app", mustReq("50m", "32Mi")),
				},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "web-1", Namespace: "ns", Labels: map[string]string{"app": "web"},
				OwnerReferences: []metav1.OwnerReference{{Kind: "ReplicaSet", Name: "web-abc", Controller: &truthy}},
			},
			Spec: podSpecWithContainer("app", corev1.ResourceRequirements{}),
		},
		&batchv1.CronJob{
			ObjectMeta: metav1.ObjectMeta{Name: "nightly", Namespace: "ns"},
			Spec: batchv1.CronJobSpec{
				Schedule: "0 0 30 2 *",
				JobTemplate: batchv1.JobTemplateSpec{
					Spec: batchv1.JobSpec{Template: corev1.PodTemplateSpec{Spec: podSpecWithContainer("app", mustReq("5m", "8Mi"))}},
				},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "bare-pod", Namespace: "ns"},
			Spec:       podSpecWithContainer("app", mustReq("40m", "28Mi")),
		},
	)

	var buf bytes.Buffer
	if err := runList(ctx, cs, nil, "ns", "table", false, &buf); err != nil {
		t.Fatal(err)
	}
	out := buf.String()

	for _, want := range []string{"Deployment", "web", "CronJob", "nightly", "Pod", "bare-pod"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected table output to contain %q, got:\n%s", want, out)
		}
	}
	// web-1 is owned by a ReplicaSet, so it must NOT show up as a bare Pod row.
	if strings.Count(out, "web-1") != 0 {
		t.Fatalf("expected owned pod web-1 to be excluded from bare-pod listing, got:\n%s", out)
	}

	buf.Reset()
	if err := runList(ctx, cs, nil, "ns", "json", false, &buf); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `"kind": "CronJob"`) {
		t.Fatalf("expected json output to include the CronJob row, got:\n%s", buf.String())
	}
}

func TestRunListUnsupportedFormat(t *testing.T) {
	ctx := context.Background()
	cs := fake.NewSimpleClientset()
	var buf bytes.Buffer
	err := runList(ctx, cs, nil, "ns", "xml", false, &buf)
	if err == nil {
		t.Fatal("expected an error for an unsupported output format")
	}
}
