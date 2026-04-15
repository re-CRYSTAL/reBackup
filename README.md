# rebackup

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

## ✨ О проекте

`rebackup` — это быстрый и безопасный CLI-инструмент для резервного копирования и восстановления данных через `.tar.gz`.

Создан с упором на:
- ⚡ производительность
- 🔒 безопасность
- 🧱 минимальные зависимости (один бинарник)

---

## 🚀 Quick Start

```bash
# Установка
go install github.com/re-CRYSTAL/reBackup@latest

# Бэкап
rebackup backup --path /home/user/data

# Восстановление
rebackup restore --file backup.tar.gz --target /restore/path
🔥 Возможности
📦 Backup в .tar.gz с прогрессбаром
♻️ Безопасный restore (защита от path traversal)
📂 Просмотр архива без распаковки
📡 Отправка в Telegram
📊 Структурированные логи
🧠 Потоковая обработка (без роста RAM)
🧩 Один статический бинарник
🧱 Архитектура
rebackup/
├── cmd/           # CLI команды (Cobra)
├── internal/
│   ├── backup/   # Логика создания архива
│   ├── restore/  # Восстановление
│   └── security/ # Безопасность путей
├── pkg/logger/   # Логирование + прогрессбар
└── main.go
⚙️ Установка
Сборка
git clone https://github.com/re-CRYSTAL/reBackup.git
cd rebackup

make build
sudo make install
🧪 Использование
Backup
rebackup backup [flags]

-p, --path       Путь к данным (обязательно)
-o, --output     Куда сохранить архив
    --telegram   Отправить в Telegram
Примеры
rebackup backup --path /data

rebackup backup --path /data --output /backups

rebackup backup --path /data --telegram
Restore
rebackup restore [flags]

-f, --file       Архив (обязательно)
-t, --target     Куда восстановить
-l, --list       Только просмотр
Примеры
rebackup restore --file backup.tar.gz --target /restore

rebackup restore --file backup.tar.gz --list
🔐 Безопасность
✅ Защита от ../../ (Zip Slip)
✅ Проверка всех путей перед записью
✅ Ограничение чтения (10 GiB)
✅ Ошибки архивов не игнорируются
🌐 Telegram интеграция
export TELEGRAM_TOKEN="your_token"
export TELEGRAM_CHAT_ID="your_chat_id"

rebackup backup --path /data --telegram
📦 Сборка (production)
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
go build -ldflags="-s -w" -o rebackup .
📊 Пример вывода
[INFO] Backup start source="/data"
[INFO] Creating archive: backup_2024-01-15.tar.gz

Backup [████████████████████████████████] 100%

✅ Backup created
🧭 Roadmap
 Инкрементальные бэкапы
 Шифрование архивов
 S3 / MinIO поддержка
 Конфиг-файл (YAML)
 TUI интерфейс
🤝 Contributing

Pull requests приветствуются. Для крупных изменений сначала открой issue.

📄 License

GPL-3.0
