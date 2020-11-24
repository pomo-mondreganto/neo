package cmd

import (
	"context"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"neo/cmd/client/cli"
	"neo/internal/client"
)

// addCmd represents the add command
var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add an exploit",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg := client.MustUnmarshalConfig()
		cli := cli.NewAdd(cmd, args, cfg)
		ctx := context.Background()
		if err := cli.Run(ctx); err != nil {
			logrus.Fatalf("Error: %v", err)
		}
		logrus.Debugf("Add finished")
	},
}

func init() {
	rootCmd.AddCommand(addCmd)

	addCmd.PersistentFlags().String("name", "", "exploit name")
	addCmd.PersistentFlags().BoolP("dir", "d", false, "add exploit as a directory")
}
