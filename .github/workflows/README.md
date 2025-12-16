# GitHub Actions Workflows

## Release Workflow

The `release.yml` workflow automatically builds and releases QMLauncher when a new tag is pushed.

### How it works

1. **Trigger**: Pushes to tags matching `v*` (e.g., `v1.0.3`, `v1.0.3-rc1`)
2. **Build**: Builds launcher for Windows, Linux, and macOS with code obfuscation
3. **Release**: Creates a GitHub release with:
   - Release notes extracted from `CHANGELOG.md`
   - Built binaries for all platforms
   - Automatic prerelease flag for `-rc`, `-alpha`, or `-beta` versions

### Creating a release

1. Update `CHANGELOG.md` and `CHANGELOG.ru.md` with new version section
2. Update `wails.json` with new `productVersion`
3. Commit changes:
   ```bash
   git add CHANGELOG.md CHANGELOG.ru.md wails.json
   git commit -m "chore: prepare release v1.0.3"
   ```
4. Create and push tag:
   ```bash
   git tag -a v1.0.3 -m "Release v1.0.3"
   git push origin v1.0.3
   ```
5. GitHub Actions will automatically:
   - Build binaries for all platforms
   - Extract release notes from CHANGELOG
   - Create GitHub release with artifacts

### Release notes format

The workflow extracts release notes from `CHANGELOG.md` by finding the section matching the version tag (without `v` prefix). For example, tag `v1.0.3-rc1` will extract notes from `## [1.0.3-rc1]` section.

### Prereleases

Tags containing `-rc`, `-alpha`, or `-beta` will be marked as prereleases automatically.

