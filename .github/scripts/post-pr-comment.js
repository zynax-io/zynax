// SPDX-License-Identifier: Apache-2.0
// Upsert a marker-tagged comment on a pull request.
//
// Called only from the trusted pr-comment.yml workflow (workflow_run), which
// runs from the base repo's default branch with a write-scoped token and never
// checks out or executes PR/fork code. The body is read from a file produced by
// the untrusted CI run and is treated as opaque data; this script is the single
// place a write token touches the issue-comment API (#1668).

'use strict';

const fs = require('fs');

module.exports = async function postPrComment({ github, context, core, issueNumber, bodyPath, marker }) {
  if (!fs.existsSync(bodyPath)) {
    core.info(`No comment body at ${bodyPath} — skipping.`);
    return;
  }
  const body = fs.readFileSync(bodyPath, 'utf8');

  const { data: comments } = await github.rest.issues.listComments({
    owner: context.repo.owner,
    repo:  context.repo.repo,
    issue_number: issueNumber,
  });
  const existing = comments.find(c => c.body && c.body.includes(marker));

  if (existing) {
    await github.rest.issues.updateComment({
      owner: context.repo.owner, repo: context.repo.repo,
      comment_id: existing.id, body,
    });
    core.info(`Updated comment ${existing.id} on #${issueNumber}.`);
  } else {
    await github.rest.issues.createComment({
      owner: context.repo.owner, repo: context.repo.repo,
      issue_number: issueNumber, body,
    });
    core.info(`Created comment on #${issueNumber}.`);
  }
};
