package cmd

import (
	"context"
	"fmt"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"path/filepath"
)

var RootCmd = &cobra.Command{
	Use:   "resource",
	Short: "Kubernetes resource viewer",
}

func Execute(ctx context.Context) {
	cobra.CheckErr(RootCmd.ExecuteContext(ctx))
}

func listDeployments(ctx context.Context, clientset kubernetes.Interface, namespace string, table *tablewriter.Table) {
	deployments, _ := clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	for _, deployment := range deployments.Items {
		for _, container := range deployment.Spec.Template.Spec.Containers {
			reqCPU := container.Resources.Requests.Cpu().MilliValue()
			reqMem := container.Resources.Requests.Memory().Value() / (1024 * 1024) // Convert to Mi
			limCPU := container.Resources.Limits.Cpu().MilliValue()
			limMem := container.Resources.Limits.Memory().Value() / (1024 * 1024) // Convert to Mi
			table.Append([]string{"Deployment", deployment.Name, container.Name, fmt.Sprintf("%vm", reqCPU), fmt.Sprintf("%vMi", reqMem), fmt.Sprintf("%vm", limCPU), fmt.Sprintf("%vMi", limMem)})
		}
	}
}

func listStatefulSets(ctx context.Context, clientset kubernetes.Interface, namespace string, table *tablewriter.Table) {
	statefulSets, _ := clientset.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
	for _, statefulSet := range statefulSets.Items {
		for _, container := range statefulSet.Spec.Template.Spec.Containers {
			reqCPU := container.Resources.Requests.Cpu().MilliValue()
			reqMem := container.Resources.Requests.Memory().Value() / (1024 * 1024) // Convert to Mi
			limCPU := container.Resources.Limits.Cpu().MilliValue()
			limMem := container.Resources.Limits.Memory().Value() / (1024 * 1024) // Convert to Mi
			table.Append([]string{"StatefulSet", statefulSet.Name, container.Name, fmt.Sprintf("%vm", reqCPU), fmt.Sprintf("%vMi", reqMem), fmt.Sprintf("%vm", limCPU), fmt.Sprintf("%vMi", limMem)})
		}
	}
}

func listDaemonSets(ctx context.Context, clientset kubernetes.Interface, namespace string, table *tablewriter.Table) {
	daemonSets, _ := clientset.AppsV1().DaemonSets(namespace).List(ctx, metav1.ListOptions{})
	for _, daemonSet := range daemonSets.Items {
		for _, container := range daemonSet.Spec.Template.Spec.Containers {
			reqCPU := container.Resources.Requests.Cpu().MilliValue()
			reqMem := container.Resources.Requests.Memory().Value() / (1024 * 1024) // Convert to Mi
			limCPU := container.Resources.Limits.Cpu().MilliValue()
			limMem := container.Resources.Limits.Memory().Value() / (1024 * 1024) // Convert to Mi
			table.Append([]string{"DaemonSet", daemonSet.Name, container.Name, fmt.Sprintf("%vm", reqCPU), fmt.Sprintf("%vMi", reqMem), fmt.Sprintf("%vm", limCPU), fmt.Sprintf("%vMi", limMem)})
		}
	}
}

func listSpecificDeployment(ctx context.Context, clientset kubernetes.Interface, namespace, deploymentName string, table *tablewriter.Table) {
	deployment, _ := clientset.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
	for _, container := range deployment.Spec.Template.Spec.Containers {
		reqCPU := container.Resources.Requests.Cpu().MilliValue()
		reqMem := container.Resources.Requests.Memory().Value() / (1024 * 1024) // Convert to Mi
		limCPU := container.Resources.Limits.Cpu().MilliValue()
		limMem := container.Resources.Limits.Memory().Value() / (1024 * 1024) // Convert to Mi
		table.Append([]string{"Deployment", deployment.Name, container.Name, fmt.Sprintf("%vm", reqCPU), fmt.Sprintf("%vMi", reqMem), fmt.Sprintf("%vm", limCPU), fmt.Sprintf("%vMi", limMem)})
	}
}

func GetCurrentNamespace() (string, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	kubeconfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
	namespace, _, err := kubeconfig.Namespace()
	if err != nil {
		return "", err
	}
	return namespace, nil
}

func NewClientLocal() (kubernetes.Interface, error) {
	config, err := KubeConfig()
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}

func KubeConfig() (*rest.Config, error) {
	kubeconfigPath := os.Getenv("KUBECONFIG")
	if kubeconfigPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		kubeconfigPath = filepath.Join(homeDir, ".kube", "config")
	}
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, err
	}
	return config, nil
}
