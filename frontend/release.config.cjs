/** @type {import('semantic-release').Options} */
module.exports = {
  branches: ['main'],
  plugins: [
    '@semantic-release/commit-analyzer',
    '@semantic-release/release-notes-generator',
    ['@semantic-release/changelog', {changelogFile: '../CHANGELOG.md'}],
    ['@semantic-release/npm', {npmPublish: false}],
    [
      '@semantic-release/git',
      {
        assets: [
          '../CHANGELOG.md',
          '../CHANGELOG_EN.md',
          'package.json',
          'package-lock.json',
        ],
        message: 'chore(release): ${nextRelease.version} [skip ci]',
      },
    ],
    ['@semantic-release/exec', {publishCmd: 'git push origin main --follow-tags'}],
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
