module.exports = {
    branches: ['main'],
    plugins: [
      '@semantic-release/commit-analyzer',
      '@semantic-release/release-notes-generator',
      '@semantic-release/changelog',
      '@semantic-release/git',
      [
        '@semantic-release/exec',
        {
          prepareCmd: 'echo ${nextRelease.version} > version.txt',
        },
      ],
      '@semantic-release/github'
    ],
    preset: 'conventionalcommits',
    tagFormat: 'v${version}',
  };