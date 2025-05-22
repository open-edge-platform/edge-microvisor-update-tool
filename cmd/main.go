package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	apply "os.abupdate.tool/internal/apply"
	commit "os.abupdate.tool/internal/commit"
	rollback "os.abupdate.tool/internal/rollback"
	write "os.abupdate.tool/internal/write"
)

var (
	debug bool
)

var rootCmd = &cobra.Command{
	Use:   "os-update-tool",
	Short: "os-update-tool ver-2.4",
	Long:  `Usage: sudo os-update-tool [command] [flags]`,
}

var writeCmd = &cobra.Command{
	Use:   "write [update-image-path] [checksum]",
	Short: "Write rootfs partition",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		updateImagePath := args[0]
		checksumValue := args[1]
		fmt.Println(write.WritePartition(updateImagePath, checksumValue))
	},
}

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply updated image as next boot",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(apply.ApplyChange())
	},
}

var commitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Commit updated image as default boot",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(commit.CommitChange())
	},
}

var rollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "Restore to previous boot",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(rollback.RollbackChange())
	},
}

var displayCmd = &cobra.Command{
	Use:   "display",
	Short: "Display current active partition",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Displaying current active partition...")
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug mode")

	// Add commands in the desired order
	rootCmd.AddCommand(writeCmd)
	rootCmd.AddCommand(applyCmd)
	rootCmd.AddCommand(commitCmd)
	rootCmd.AddCommand(rollbackCmd)
	rootCmd.AddCommand(displayCmd)
}

func main() {

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

}
