package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// version 通过 -ldflags 注入。
var version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "打印版本信息",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("ikuai-cli %s\n", version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
