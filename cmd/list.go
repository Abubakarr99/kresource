package cmd

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

var cmdList = &cobra.Command{
	Use:   "list",
	Short: "Lists resource requests, limits, and optionally actual usage",
	Long: `Lists CPU/memory requests and limits for Deployments, StatefulSets,
DaemonSets, standalone ReplicaSets, standalone Jobs, CronJobs, and unmanaged
Pods in a namespace. Pass --with-usage to also show actual usage averaged
across each workload's replicas, from metrics-server.`,
	Run: func(cmd *cobra.Command, args []string) {
		fs := cmd.Flags()
		ctx := context.Background()

		clientset, err := NewClientLocal()
		if err != nil {
			log.Fatalln(err)
		}

		namespace, _ := fs.GetString("namespace")
		if namespace == "" {
			namespace, err = GetCurrentNamespace()
			if err != nil {
				log.Fatal(err)
			}
		}

		outputFormat, _ := fs.GetString("output")
		withUsage, _ := fs.GetBool("with-usage")

		var metricsClient metricsv.Interface
		if withUsage {
			mc, mErr := NewMetricsClientLocal()
			if mErr != nil {
				fmt.Fprintln(os.Stderr, "warning: could not create metrics client, continuing without usage data:", mErr)
			} else {
				metricsClient = mc
			}
		}

		if err := runList(ctx, clientset, metricsClient, namespace, outputFormat, withUsage, os.Stdout); err != nil {
			log.Fatalln(err)
		}
	},
}

// runList holds everything after client/flag setup: collecting rows from
// every workload kind, optionally augmenting with metrics-server usage,
// computing totals, and rendering. Split out from cmdList.Run so it can be
// exercised with fake/injected clients instead of a real cluster.
func runList(ctx context.Context, clientset kubernetes.Interface, metricsClient metricsv.Interface, namespace, outputFormat string, withUsage bool, w io.Writer) error {
	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	var podMetrics map[string]map[string]podContainerUsage
	if withUsage && metricsClient != nil {
		pm, fErr := fetchPodMetrics(ctx, metricsClient, namespace)
		if fErr != nil {
			fmt.Fprintln(os.Stderr, "warning: metrics-server unavailable, continuing without usage data:", fErr)
		} else {
			podMetrics = pm
		}
	}

	var rows []ResourceRow
	collectors := []func() ([]ResourceRow, error){
		func() ([]ResourceRow, error) {
			return collectDeployments(ctx, clientset, namespace, pods.Items, podMetrics)
		},
		func() ([]ResourceRow, error) {
			return collectStatefulSets(ctx, clientset, namespace, pods.Items, podMetrics)
		},
		func() ([]ResourceRow, error) {
			return collectDaemonSets(ctx, clientset, namespace, pods.Items, podMetrics)
		},
		func() ([]ResourceRow, error) {
			return collectReplicaSets(ctx, clientset, namespace, pods.Items, podMetrics)
		},
		func() ([]ResourceRow, error) {
			return collectJobs(ctx, clientset, namespace, pods.Items, podMetrics)
		},
		func() ([]ResourceRow, error) {
			return collectCronJobs(ctx, clientset, namespace)
		},
	}
	for _, collect := range collectors {
		r, err := collect()
		if err != nil {
			return err
		}
		rows = append(rows, r...)
	}
	rows = append(rows, collectBarePods(pods.Items, podMetrics)...)

	totals := computeTotals(rows)

	switch outputFormat {
	case "table", "":
		renderTable(w, rows, totals, withUsage)
		return nil
	case "json":
		return renderJSON(w, rows, totals)
	case "yaml":
		return renderYAML(w, rows, totals)
	case "csv":
		return renderCSV(w, rows, withUsage)
	default:
		return fmt.Errorf("unsupported output format %q (want table, json, yaml, or csv)", outputFormat)
	}
}

func init() {
	RootCmd.AddCommand(cmdList)
	cmdList.Flags().StringP("namespace", "n", "", "The namespace of the resources")
	cmdList.Flags().StringP("output", "o", "table", "Output format: table, json, yaml, csv")
	cmdList.Flags().BoolP("with-usage", "u", false, "Include actual CPU/memory usage from metrics-server")
}
