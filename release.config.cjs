/** @type {import('semantic-release').Options} */
// Loaded with repository root as semantic-release cwd so @semantic-release/git sees all modified paths.
module.exports = {
  branches: ['main'],
  plugins: [
    [
      '@semantic-release/commit-analyzer',
      {
        // Extra rules run first; if none match, default rules still apply (feat/fix/perf, …).
        // Without this, Dependabot-style `chore(deps):` merges do not emit a new version.
        releaseRules: [
          { type: 'chore', scope: 'deps', release: 'patch' },
          { type: 'chore', scope: 'deps-dev', release: 'patch' },
          { type: 'build', scope: 'deps', release: 'patch' },
        ],
      },
    ],
    '@semantic-release/release-notes-generator',
    ['@semantic-release/changelog', { changelogFile: 'CHANGELOG.md' }],
    ['@semantic-release/npm', { npmPublish: false, pkgRoot: 'frontend' }],
    [
      '@semantic-release/exec',
      {
        prepareCmd:
          'node frontend/scripts/sync-go-version.mjs "<%= nextRelease.version %>"',
        publishCmd: 'git push origin main --follow-tags',
      },
    ],
    [
      '@semantic-release/git',
      {
        assets: [
          'CHANGELOG.md',
          'CHANGELOG_EN.md',
          'version.go',
          'internal/version/version.go',
          'frontend/package.json',
          'frontend/package-lock.json',
        ],
        // No [skip ci]: tag push must run `.github/workflows/release-qmlauncher.yml`
        // so GitHub Release gets Linux/Windows binaries (softprops upload).
        message: 'chore(release): ${nextRelease.version}',
      },
    ],
    // Do not use @semantic-release/github here: it creates a release without build artifacts.
    // The tag push triggers `.github/workflows/release-qmlauncher.yml`, which attaches binaries.
  ],
}
