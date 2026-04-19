/** @type {import('semantic-release').Options} */
module.exports = {
  branches: ['main'],
  plugins: [
    '@semantic-release/commit-analyzer',
    '@semantic-release/release-notes-generator',
    ['@semantic-release/changelog', {changelogFile: '../CHANGELOG.md'}],
    ['@semantic-release/npm', {npmPublish: false}],
    [
      '@semantic-release/exec',
      {
        prepareCmd:
          'node ./scripts/sync-go-version.mjs "<%= nextRelease.version %>"',
        publishCmd: 'git push origin main --follow-tags',
      },
    ],
    [
      '@semantic-release/git',
      {
        assets: [
          '../CHANGELOG.md',
          '../CHANGELOG_EN.md',
          '../version.go',
          '../internal/version/version.go',
          'package.json',
          'package-lock.json',
        ],
        message: 'chore(release): ${nextRelease.version} [skip ci]',
      },
    ],
    [
      '@semantic-release/github',
      {
        successCommentCondition: false,
        failCommentCondition: false,
        releasedLabels: false,
      },
    ],
  ],
};
