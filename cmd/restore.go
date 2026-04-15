package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"rebackup/internal/restore"
	"rebackup/pkg/logger"
)

var (
	restoreFile   string
	restoreTarget string
	restoreList   bool
)

var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore files from a .tar.gz backup archive",
	Long: `Restore files from a .tar.gz backup archive to the specified directory.

Path traversal protection is always active — entries that would escape the
target directory are automatically skipped and reported.

Examples:
  rebackup restore --file backup_2024-01-15_10-30.tar.gz --target /home/user/restore
  rebackup restore --file backup.tar.gz --list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		log := logger.New()

		r := restore.New(log)

		// List-only mode: no --target required
		if restoreList {
			return r.ListContents(restoreFile)
		}

		if restoreTarget == "" {
			return fmt.Errorf("--target is required (use --list to only inspect the archive)")
		}

		if err := r.Restore(restoreFile, restoreTarget); err != nil {
			return fmt.Errorf("restore failed: %w", err)
		}

		fmt.Printf("\n✅ Restore completed → %s\n", restoreTarget)
		return nil
	},
}

func init() {
	restoreCmd.Flags().StringVarP(&restoreFile, "file", "f", "", "Path to backup archive (required)")
	restoreCmd.Flags().StringVarP(&restoreTarget, "target", "t", "", "Destination directory for extracted files")
	restoreCmd.Flags().BoolVarP(&restoreList, "list", "l", false, "List archive contents without extracting")

	_ = restoreCmd.MarkFlagRequired("file")
}
