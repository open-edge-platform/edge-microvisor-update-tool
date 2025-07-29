package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	apply "os.update.tool/internal/apply"
	commit "os.update.tool/internal/commit"
	rollback "os.update.tool/internal/rollback"
	write "os.update.tool/internal/write"
	core "os.update.tool/pkg/core"
	"os.update.tool/pkg/logger"
)

var (
	debug   bool
	Version string
)

func printVersion() {
	versionFile := "VERSION"
	versionLocal, err := os.ReadFile(versionFile)
	if err != nil {
		versionLocal = []byte(Version)
	}
	fmt.Printf("os-update-tool version: %s\n", string(versionLocal))
}

var rootCmd = &cobra.Command{
	Use:   "os-update-tool",
	Short: "os-update-tool ver-3.0",
	Long:  `Usage: sudo os-update-tool [command] [flags]`,
}

var writeCmd = &cobra.Command{
	Use:   "write [update-image-path] [checksum]",
	Short: "Write rootfs partition",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		updateImagePath := args[0]
		checksumValue := args[1]
		devMode, _ := cmd.Flags().GetBool("dev")
		if devMode {
			logger.LogInfo("Development mode enabled")
		}
		err := write.WritePartition(updateImagePath, checksumValue, devMode)
		if err != nil {
			logger.LogError("Error writing partition: %v", err)
			return err
		}
		return nil
	},
}

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply updated image as next boot",
	RunE: func(cmd *cobra.Command, args []string) error {
		err := apply.ApplyChange()
		if err != nil {
			logger.LogError("Error applying new OS: %v", err)
			return err
		}
		return nil
	},
}

var commitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Commit updated image as default boot",
	RunE: func(cmd *cobra.Command, args []string) error {
		err := commit.CommitChange()
		if err != nil {
			logger.LogError("Error committing new OS: %v", err)
			return err
		}
		return nil
	},
}

var rollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "Restore to previous boot",
	RunE: func(cmd *cobra.Command, args []string) error {
		err := rollback.RollbackChange()
		if err != nil {
			logger.LogError("Error rolling back to previous boot: %v", err)
			return err
		}
		return nil
	},
}

var displayCmd = &cobra.Command{
	Use:   "display",
	Short: "Display current active partition",
	RunE: func(cmd *cobra.Command, args []string) error {
		logger.LogInfo("Displaying current active partition...")
		_, err := core.GetActivePartition()
		if err != nil {
			logger.LogError("Error getting current active partition: %v", err)
			return err
		}
		return nil
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

	writeCmd.Flags().Bool("dev", false, "Enable development mode")
}

func main() {
	printVersion()
	if err := rootCmd.Execute(); err != nil {
		logger.LogError("Command execution failed: %v", err)
		os.Exit(1)
	}
}
