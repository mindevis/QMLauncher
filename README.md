# QMLauncher

Десктоп-приложение в **[QMProject](https://github.com/mindevis/QMProject)** (`desktop/QMLauncher`). **[`README.md`](../../README.md#monorepo-layout)**, **[`docs/github-push-releases-and-packages.md`](../../docs/github-push-releases-and-packages.md)**.

Кроссплатформенный **лаунчер Minecraft** с GUI на **Wails** (Go + React/TypeScript): инстансы, модлоадеры, MSA/Mojang, интеграция с **QMServer**.

**Платформа:** десктоп — **Linux x86_64**, **Windows x86_64**, **macOS** (amd64 / arm64). Сборка даёт **нативный исполняемый файл** со встроенным фронтендом.

## Документация

См. этот README, **`services/QMServer/README.md`** (API) и корневой **`README.md`** (деплой).

## Сборка

Версия берётся из **`VERSION`** и пробрасывается в бинарник (`-ldflags`); при сборке в CI — из тега релиза.

```bash
cd frontend && npm ci && npm run build && cd ..
make build    # → build/QMLauncher-<os>-<arch>
```

Кросс-сборка (нужен Go; на Linux для Windows/macOS возможны ограничения **CGO** / WebView — см. **`Makefile`**):

```bash
make linux          # build/QMLauncher-linux-amd64
make windows        # build/QMLauncher-windows-amd64.exe
make macos          # build/QMLauncher-darwin-amd64
make macos-arm64    # build/QMLauncher-darwin-arm64
```

Проверка версии:

```bash
./build/QMLauncher-linux-amd64 -version
```

## Релизы (GitHub)

По тегу **`v*`** workflow **`.github/workflows/release-qmlauncher.yml`** собирает **Linux x86_64** (Ubuntu runner: зависимости GTK/WebKit для Wails, затем `npm ci` / `npm run build` во **`frontend/`** и **`make linux`**), публикует бинарник **`QMLauncher-linux-amd64`**, архив **`.tar.gz`** и SHA256. Windows/macOS в этом workflow **не** собираются — их можно собирать локально или отдельными job’ами при необходимости.

## Разработка и тесты

```bash
make check      # fmt, vet, тесты, линт (см. Makefile)
```

## Лицензия

[LICENSE](LICENSE) (MIT).
