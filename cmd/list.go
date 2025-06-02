package cmd

import (
	"context"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"log"
	"os"
)

var cmdList = &cobra.Command{
	Use:   "list",
	Short: "Lists resource request",
	Long:  `Lists resource request`,
	Run: func(cmd *cobra.Command, args []string) {
		fs := cmd.Flags()
		var namespace string
		var err error
		ctx := context.Background()
		clientset, err := NewClientLocal()
		if err != nil {
			log.Fatalln(err)
		}
		namespace, _ = fs.GetString("namespace")
		if namespace == "" {
			namespace, err = GetCurrentNamespace()
			if err != nil {
				log.Fatal(err)
			}
		}
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"Resource Type", "Resource Name", "Container", "Request CPU", "Request Memory", "Limit CPU", "Limit Memory"})
		listDeployments(ctx, clientset, namespace, table)
		listStatefulSets(ctx, clientset, namespace, table)
		listDaemonSets(ctx, clientset, namespace, table)
		table.Render()
	},
}

func init() {
	RootCmd.AddCommand(cmdList)
	cmdList.Flags().StringP("namespace", "n", "", "The namespace of the resources")
}
