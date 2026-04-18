# QMLauncher

Part of **[QMProject](https://github.com/mindevis/QMProject)** (`desktop/QMLauncher`). **[`README.md`](../../README.md#monorepo-layout)**, **[`docs/github-push-releases-and-packages.md`](../../docs/github-push-releases-and-packages.md)**.

Cross-platform **Minecraft launcher** with a **Wails** desktop UI (Go + React/TypeScript): instances, mod loaders, MSA/Mojang, optional **QMServer** integration.

**Platforms:** desktop — **Linux x86_64**, **Windows x86_64**, **macOS** (amd64 / arm64). Builds produce a **native binary** with the embedded frontend.

## Documentation

See this README, **`services/QMServer/README.md`**, and the root **`README.md`** for deployment.

## Build

Version is read from **`VERSION`** and linked via `-ldflags`; CI uses the release tag.

```bash
cd frontend && npm ci && npm run build && cd ..
make build    # → build/QMLauncher-<os>-<arch>
```

Cross-build (Go required; **CGO** / WebView limits may apply for Windows/macOS from Linux — see **`Makefile`**):

```bash
make linux
make windows
make macos
```

```bash
./build/QMLauncher-linux-amd64 -version
```

## GitHub Releases

Tag **`v*`** triggers **`.github/workflows/release-qmlauncher.yml`**, which builds **Linux x86_64** (GTK/WebKit deps, `npm ci` + `npm run build` in **`frontend/`**, then **`make linux`**) and uploads **`QMLauncher-linux-amd64`**, a **`.tar.gz`**, and SHA256 checksums. Windows/macOS binaries are **not** produced in this workflow — build locally or extend CI if needed.

## License

[LICENSE](LICENSE) (MIT).
