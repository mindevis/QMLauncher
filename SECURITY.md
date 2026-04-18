# QMLauncher security notes

## Local tokens and cloud accounts

- **Cloud account tokens** (QMServer JWT) and **CurseForge keys** may be stored under the user profile (e.g. `~/.qmlauncher/`). Protect the home directory permissions; on shared machines use separate OS accounts.
- Optional env **`CURSEFORGE_API_KEY`** overrides file settings — keep the environment private.

## Debug logging

- Verbose HTTP debug logs **redact** tokens and sensitive headers (see `internal/network/debug_transport.go` and `internal/debuglog`). **Do not enable** verbose debug on untrusted machines or when sharing log files.
- Avoid logging **PII** (email, UUIDs) in custom patches; keep logs for diagnostics only.

## Wails / IPC surface

- Only methods on the **`App`** struct bound in `main.go` are exposed to the WebView. Treat every bound method as a trusted local API: validate inputs, never pass secrets to the frontend unnecessarily, and keep the embedded **`frontend/dist`** free of remote script injection (supply chain / build pipeline).

## Updates

Prefer **signed** or **checksum-verified** update channels from your official release host; verify publisher when installing new binaries.

- **GitHub Releases:** для каждого вложения обычно публикуется **`*.sha256`** — сверяйте **`sha256sum`** перед заменой бинарника.

### Цепочка обновлений (углублённый аудит, чеклист)

- **Источник:** только релизы из вашего **официального** GitHub / CDN; сравнивайте организацию и имя репозитория с документацией.
- **`--upgrade`:** бинарь и unit-файлы перезаписываются с GitHub Releases — убедитесь, что на хосте нет сторонних правок в **`/opt/qmclient`**, которые вы не хотите потерять; делайте бэкап перед обновлением.
- **Wails / WebView:** встроенный фронт — **`frontend/dist`** из вашей сборки; не подменяйте артефакты неизвестными архивами.
- **IPC:** методы **`App`** в **`main.go`** — единственная доверенная граница для UI; при добавлении новых вызовов проверяйте аргументы и отсутствие утечки локальных путей/токенов в лог UI.
