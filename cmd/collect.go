package cmd

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func cpuMilliValue(rl corev1.ResourceList) (int64, bool) {
	q, ok := rl[corev1.ResourceCPU]
	if !ok {
		return 0, false
	}
	return q.MilliValue(), true
}

func memMiValue(rl corev1.ResourceList) (int64, bool) {
	q, ok := rl[corev1.ResourceMemory]
	if !ok {
		return 0, false
	}
	return q.Value() / (1024 * 1024), true
}

func ownedByKind(owners []metav1.OwnerReference, kind string) bool {
	for _, o := range owners {
		if o.Kind == kind {
			return true
		}
	}
	return false
}

func rowsFromContainers(kind, name string, containers []corev1.Container, init bool, usage map[string]containerUsage) []ResourceRow {
	rows := make([]ResourceRow, 0, len(containers))
	for _, c := range containers {
		reqCPU, _ := cpuMilliValue(c.Resources.Requests)
		reqMem, _ := memMiValue(c.Resources.Requests)
		limCPU, limCPUSet := cpuMilliValue(c.Resources.Limits)
		limMem, limMemSet := memMiValue(c.Resources.Limits)

		row := ResourceRow{
			Kind: kind, Name: name, Container: c.Name, Init: init,
			ReqCPU:    reqCPU,
			ReqMem:    reqMem,
			LimCPU:    limCPU,
			LimCPUSet: limCPUSet,
			LimMem:    limMem,
			LimMemSet: limMemSet,
		}

		if u, ok := usage[c.Name]; ok && u.count > 0 {
			avgCPU := u.cpuMilliSum / int64(u.count)
			avgMem := u.memMiSum / int64(u.count)
			row.ActualCPU = &avgCPU
			row.ActualMem = &avgMem
		}

		rows = append(rows, row)
	}
	return rows
}

func rowsFromPodSpec(kind, name string, spec corev1.PodSpec, usage map[string]containerUsage) []ResourceRow {
	rows := rowsFromContainers(kind, name, spec.InitContainers, true, usage)
	rows = append(rows, rowsFromContainers(kind, name, spec.Containers, false, usage)...)
	return rows
}

func collectDeployments(ctx context.Context, clientset kubernetes.Interface, namespace string, pods []corev1.Pod, podMetrics map[string]map[string]podContainerUsage) ([]ResourceRow, error) {
	list, err := clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var rows []ResourceRow
	for _, d := range list.Items {
		usage := aggregateUsage(pods, podMetrics, selectorFor(d.Spec.Selector))
		rows = append(rows, rowsFromPodSpec("Deployment", d.Name, d.Spec.Template.Spec, usage)...)
	}
	return rows, nil
}

func collectStatefulSets(ctx context.Context, clientset kubernetes.Interface, namespace string, pods []corev1.Pod, podMetrics map[string]map[string]podContainerUsage) ([]ResourceRow, error) {
	list, err := clientset.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var rows []ResourceRow
	for _, s := range list.Items {
		usage := aggregateUsage(pods, podMetrics, selectorFor(s.Spec.Selector))
		rows = append(rows, rowsFromPodSpec("StatefulSet", s.Name, s.Spec.Template.Spec, usage)...)
	}
	return rows, nil
}

func collectDaemonSets(ctx context.Context, clientset kubernetes.Interface, namespace string, pods []corev1.Pod, podMetrics map[string]map[string]podContainerUsage) ([]ResourceRow, error) {
	list, err := clientset.AppsV1().DaemonSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var rows []ResourceRow
	for _, d := range list.Items {
		usage := aggregateUsage(pods, podMetrics, selectorFor(d.Spec.Selector))
		rows = append(rows, rowsFromPodSpec("DaemonSet", d.Name, d.Spec.Template.Spec, usage)...)
	}
	return rows, nil
}

// collectReplicaSets only reports ReplicaSets that aren't owned by a
// Deployment, since a Deployment-owned one is already covered via the
// Deployment itself and would otherwise double-count the same containers.
func collectReplicaSets(ctx context.Context, clientset kubernetes.Interface, namespace string, pods []corev1.Pod, podMetrics map[string]map[string]podContainerUsage) ([]ResourceRow, error) {
	list, err := clientset.AppsV1().ReplicaSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var rows []ResourceRow
	for _, rs := range list.Items {
		if ownedByKind(rs.OwnerReferences, "Deployment") {
			continue
		}
		usage := aggregateUsage(pods, podMetrics, selectorFor(rs.Spec.Selector))
		rows = append(rows, rowsFromPodSpec("ReplicaSet", rs.Name, rs.Spec.Template.Spec, usage)...)
	}
	return rows, nil
}

// collectJobs only reports Jobs that aren't owned by a CronJob, since a
// CronJob-owned one is already covered via the CronJob's job template.
func collectJobs(ctx context.Context, clientset kubernetes.Interface, namespace string, pods []corev1.Pod, podMetrics map[string]map[string]podContainerUsage) ([]ResourceRow, error) {
	list, err := clientset.BatchV1().Jobs(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var rows []ResourceRow
	for _, j := range list.Items {
		if ownedByKind(j.OwnerReferences, "CronJob") {
			continue
		}
		usage := aggregateUsage(pods, podMetrics, selectorFor(j.Spec.Selector))
		rows = append(rows, rowsFromPodSpec("Job", j.Name, j.Spec.Template.Spec, usage)...)
	}
	return rows, nil
}

// collectCronJobs reports the job template's configured resources. There's
// no live selector to sample usage against between runs, so actual usage is
// always left unset for this kind.
func collectCronJobs(ctx context.Context, clientset kubernetes.Interface, namespace string) ([]ResourceRow, error) {
	list, err := clientset.BatchV1().CronJobs(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var rows []ResourceRow
	for _, cj := range list.Items {
		rows = append(rows, rowsFromPodSpec("CronJob", cj.Name, cj.Spec.JobTemplate.Spec.Template.Spec, nil)...)
	}
	return rows, nil
}

// collectBarePods reports pods with no OwnerReferences at all, i.e. created
// directly rather than by any controller this tool already covers above.
func collectBarePods(pods []corev1.Pod, podMetrics map[string]map[string]podContainerUsage) []ResourceRow {
	var rows []ResourceRow
	for _, p := range pods {
		if len(p.OwnerReferences) > 0 {
			continue
		}
		usage := usageForPod(podMetrics, p.Name)
		rows = append(rows, rowsFromPodSpec("Pod", p.Name, p.Spec, usage)...)
	}
	return rows
}
