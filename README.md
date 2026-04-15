## reBackup

<p align="center">
  <b>Fast. Safe. Minimal.</b><br>
  Production-ready CLI для backup и restore в Linux
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.22+-blue?logo=go">
  <img src="https://img.shields.io/badge/platform-linux-lightgrey">
  <img src="https://img.shields.io/badge/license-GPL--3.0-green">
  <img src="https://img.shields.io/badge/status-stable-success">
</p>

---
## Features

| Feature | Details |
|---------|---------|
| **Backup** | Timestamped `.tar.gz` archives with ASCII progress bar |
| **Restore** | Safe extraction with path-traversal protection |
| **List** | Inspect archive contents without extracting |
| **Security** | Every path validated before write (Zip Slip / tar slip prevention) |
| **Logging** | Structured INFO / ERROR / DEBUG levels with timestamps |
| **Telegram** | Optional upload to Telegram chat via Bot API (≤ 50 MiB) |
| **Large dirs** | Streaming I/O — memory usage stays constant regardless of archive size |
| **Single binary** | No runtime dependencies; one `go build` produces a static binary |

---

## Project structure

```
rebackup/
├── cmd/
│   ├── root.go       # Cobra root command & version
│   ├── backup.go     # `backup` subcommand flags + wiring
│   └── restore.go    # `restore` subcommand flags + wiring
├── internal/
│   ├── backup/
│   │   └── backup.go   # Archive creation, progress tracking, Telegram upload
│   ├── restore/
│   │   └── restore.go  # Safe extraction, archive listing
│   └── security/
│       └── security.go # SafePath(), ValidateArchivePath()
├── pkg/
│   └── logger/
│       └── logger.go   # Levelled logger + ASCII progress bar
├── main.go
├── go.mod
├── Makefile
└── README.md
```

---

## Requirements

- Go **1.22+**
- Linux (Ubuntu 20.04 / Debian 11 or newer recommended)

---

## Installation

### Option A — build from source

```bash
git clone https://github.com/re-CRYSTAL/reBackup.git
cd rebackup

make build                     # produces ./rebackup
sudo make install              # copies to /usr/local/bin/rebackup
```

### Option B — go install (once pushed to GitHub)

```bash
go install github.com/re-CRYSTAL/reBackup@latest
```

The binary lands in `$(go env GOPATH)/bin`; make sure that is on your `$PATH`.

---

## Usage

### `backup`

```
rebackup backup [flags]

Flags:
  -p, --path string     Source directory to backup  (required)
  -o, --output string   Output directory for the archive  (default ".")
      --telegram        Send archive to Telegram after creation
  -h, --help            Show help
```

**Examples**

```bash
# Basic backup — archive saved in the current directory
rebackup backup --path /home/user/data

# Backup to a specific location
rebackup backup --path /home/user/data --output /mnt/backups

# Backup and push to Telegram
export TELEGRAM_TOKEN="123456:ABC-your-bot-token"
export TELEGRAM_CHAT_ID="987654321"
rebackup backup --path /home/user/data --telegram
```

Sample output:

```
[INFO]  2024/01/15 10:30:01 Backup start  source="/home/user/data"  output="."
[INFO]  2024/01/15 10:30:01 Pre-scan: 142 entries | 45.23 MiB uncompressed
[INFO]  2024/01/15 10:30:01 Creating archive: backup_2024-01-15_10-30.tar.gz
Backup     [████████████████████████████████████████] 100.0% (142/142)
[INFO]  2024/01/15 10:30:03 Backup done   archive=backup_2024-01-15_10-30.tar.gz  compressed=12.31 MiB  original=45.23 MiB

✅ Backup created: backup_2024-01-15_10-30.tar.gz
```

---

### `restore`

```
rebackup restore [flags]

Flags:
  -f, --file string     Path to backup archive  (required)
  -t, --target string   Destination directory for extracted files
  -l, --list            List archive contents without extracting
  -h, --help            Show help
```

**Examples**

```bash
# Restore to a directory
rebackup restore --file backup_2024-01-15_10-30.tar.gz --target /home/user/restored

# List archive contents without extracting
rebackup restore --file backup_2024-01-15_10-30.tar.gz --list
```

Sample `--list` output:

```
TYPE        SIZE       MODIFIED              NAME
──────────────────────────────────────────────────────────────────────
dir         0B         2024-01-15 09:15:00   data/
file        4.0KiB     2024-01-15 09:10:00   data/config.yaml
file        1.2MiB     2024-01-14 22:00:00   data/database.db
symlink     0B         2024-01-15 09:00:00   data/latest -> database.db
──────────────────────────────────────────────────────────────────────
Total: 4 entries  |  1.2MiB
```

---

## Environment variables

| Variable | Required for | Description |
|----------|-------------|-------------|
| `TELEGRAM_TOKEN` | `--telegram` | Bot token from [@BotFather](https://t.me/BotFather) |
| `TELEGRAM_CHAT_ID` | `--telegram` | Numeric chat or channel ID |

---

## Security model

`rebackup` implements **safeExtract** via `security.SafePath()`:

1. The target directory is resolved to an absolute path.
2. Every archive entry path is joined with the absolute target and cleaned (`filepath.Clean`).
3. The result is checked to confirm it has the target as a prefix.
4. Entries that escape the target (e.g. `../../etc/passwd`, `/etc/shadow`,
   `../targetDirSuffix/evil`) are **logged and skipped** — extraction continues.

Additionally:

- Each extracted file is wrapped in `io.LimitReader(10 GiB)` to prevent decompression-bomb attacks.
- Corrupted gzip/tar headers are surfaced as hard errors, not silently ignored.

---

## Building for distribution

```bash
# Statically linked binary (fully portable, no libc dependency)
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o rebackup .

# Verify it is static
file rebackup
# rebackup: ELF 64-bit LSB executable, x86-64, statically linked, stripped
```

---

## License

GPL-3.0 license
