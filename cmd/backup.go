package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"rebackup/internal/backup"
	"rebackup/pkg/logger"
)

var (
	backupPath     string
	backupOutput   string
	backupTelegram bool
)

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Create a .tar.gz backup archive from a directory",
	Long: `Create a compressed .tar.gz backup archive from the specified directory.
The archive filename is generated automatically with a timestamp:

  backup_YYYY-MM-DD_HH-MM.tar.gz

Examples:
  rebackup backup --path /home/user/data
  rebackup backup --path /home/user/data --output /mnt/backups
  rebackup backup --path /home/user/data --telegram`,
	RunE: func(cmd *cobra.Command, args []string) error {
		log := logger.New()

		// Verify the source path exists before doing any work
		if _, err := os.Stat(backupPath); os.IsNotExist(err) {
			return fmt.Errorf("source path does not exist: %s", backupPath)
		}

		b := backup.New(log)
		archivePath, err := b.Create(backupPath, backupOutput)
		if err != nil {
			return fmt.Errorf("backup failed: %w", err)
		}

		fmt.Printf("\n✅ Backup created: %s\n", archivePath)

		if backupTelegram {
			log.Info("Sending backup to Telegram…")
			if err := b.SendToTelegram(archivePath); err != nil {
				// Non-fatal: backup already exists on disk
				log.Errorf("Telegram send failed: %v", err)
				fmt.Fprintf(os.Stderr, "\n⚠️  Telegram send failed: %v\n", err)
				return nil
			}
			fmt.Println("✅ Backup sent to Telegram")
		}

		return nil
	},
}

func init() {
	backupCmd.Flags().StringVarP(&backupPath, "path", "p", "", "Source directory to backup (required)")
	backupCmd.Flags().StringVarP(&backupOutput, "output", "o", ".", "Output directory for the archive")
	backupCmd.Flags().BoolVar(&backupTelegram, "telegram", false,
		"Send archive to Telegram (needs TELEGRAM_TOKEN + TELEGRAM_CHAT_ID env vars)")

	_ = backupCmd.MarkFlagRequired("path")
}
