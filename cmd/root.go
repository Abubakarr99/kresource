package cmd

import (
	"context"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

var RootCmd = &cobra.Command{
	Use:   "resource",
	Short: "Kubernetes resource viewer",
}

func Execute(ctx context.Context) {
	cobra.CheckErr(RootCmd.ExecuteContext(ctx))
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

func NewMetricsClientLocal() (metricsv.Interface, error) {
	config, err := KubeConfig()
	if err != nil {
		return nil, err
	}
	return metricsv.NewForConfig(config)
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
