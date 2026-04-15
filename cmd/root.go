package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "rebackup",
	Short: "A reliable backup and restore utility for Linux",
	Long: `rebackup — production-ready Linux CLI utility for creating and
restoring backups using compressed .tar.gz archives.

Examples:
  rebackup backup  --path /home/user/data
  rebackup backup  --path /home/user/data --output /mnt/backups
  rebackup restore --file backup_2024-01-15_10-30.tar.gz --target /home/user/restore
  rebackup restore --file backup_2024-01-15_10-30.tar.gz --list`,
	Version: "1.0.0",
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(backupCmd)
	rootCmd.AddCommand(restoreCmd)
}
