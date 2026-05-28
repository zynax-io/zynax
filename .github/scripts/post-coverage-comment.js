// SPDX-License-Identifier: Apache-2.0
// Post (or update) the coverage report comment on a PR.
// Called from ci.yml via actions/github-script:
//   const fn = require('./.github/scripts/post-coverage-comment.js');
//   await fn({ github, context, core });

'use strict';

const fs = require('fs');
const MARKER = '<!-- zynax-coverage-report -->';

module.exports = async function postCoverageComment({ github, context, core }) {
  const commentFile = '/tmp/coverage-comment.md';
  let body;

  if (fs.existsSync(commentFile)) {
    body = fs.readFileSync(commentFile, 'utf8');
  } else {
    const runUrl = process.env.RUN_URL || '';
    body = `${MARKER}\n## Coverage Report\n\n_No coverage data available for this run._\n\n<sub>Run [#${process.env.RUN_NUMBER || '?'}](${runUrl})</sub>`;
  }

  const { data: comments } = await github.rest.issues.listComments({
    owner: context.repo.owner,
    repo:  context.repo.repo,
    issue_number: context.issue.number,
  });
  const existing = comments.find(c => c.body && c.body.includes(MARKER));

  if (existing) {
    await github.rest.issues.updateComment({
      owner: context.repo.owner, repo: context.repo.repo,
      comment_id: existing.id, body,
    });
  } else {
    await github.rest.issues.createComment({
      owner: context.repo.owner, repo: context.repo.repo,
      issue_number: context.issue.number, body,
    });
  }
};
