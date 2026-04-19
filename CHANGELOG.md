## [1.0.10](https://github.com/mindevis/qmlauncher/compare/v1.0.9...v1.0.10) (2026-04-19)


### Bug Fixes

* **ci:** dispatch Release QMLauncher after semantic-release tag ([c67cef3](https://github.com/mindevis/qmlauncher/commit/c67cef3bc579019ad68da053984b4500e2349b83))

## [1.0.9](https://github.com/mindevis/qmlauncher/compare/v1.0.8...v1.0.9) (2026-04-19)


### Bug Fixes

* **release:** treat refactor commits as patch for semantic-release ([b147c50](https://github.com/mindevis/qmlauncher/commit/b147c50b0e4b8c318660ff71c2116e3e9ee17cd9))

## [1.0.8](https://github.com/mindevis/qmlauncher/compare/v1.0.7...v1.0.8) (2026-04-19)

### Исправления

* **release:** убран `[skip ci]` из коммита релиза и отключён пустой GitHub Release от semantic-release — сборки прикрепляются workflow по тегу.
* **release:** ручной перезапуск релиза (`workflow_dispatch`) для тега `v*`.
* **feat:** периодический опрос QMServer (серверы, новости, настройки) без перезапуска лаунчера; сброс кэша `/servers` и TTL 90 с.

## [1.0.7](https://github.com/mindevis/qmlauncher/compare/v1.0.6...v1.0.7) (2026-04-19)


### Bug Fixes

* **release:** bump version on dependabot-style chore(deps) commits ([35317ff](https://github.com/mindevis/qmlauncher/commit/35317ff98f26834bed7ea5c7e74f1e1465b62f0d))

## [1.0.6](https://github.com/mindevis/qmlauncher/compare/v1.0.5...v1.0.6) (2026-04-19)


### Bug Fixes

* **qmlauncher:** run semantic-release from repo root (git plugin + pkgRoot) ([972aacd](https://github.com/mindevis/qmlauncher/commit/972aacd27da73929a4640fff1035a8bd2ab07b04))

# Changelog

Все значимые изменения **QMLauncher** в составе монорепозитория **qm-project** документируются здесь.

Формат основан на [Keep a Changelog](https://keepachangelog.com/ru/1.0.0/), версии — [SemVer](https://semver.org/lang/ru/).

## [v1.0.0] — 2026-04-17

### Добавлено

- Базовая точка журнала после переноса в монорепозиторий (`desktop/QMLauncher/`).
