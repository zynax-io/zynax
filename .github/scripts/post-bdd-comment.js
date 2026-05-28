// SPDX-License-Identifier: Apache-2.0
// Post (or update) the BDD contract test report comment on a PR.
// Called from ci.yml via actions/github-script:
//   const fn = require('./.github/scripts/post-bdd-comment.js');
//   await fn({ github, context, core });
//
// Env vars consumed: RUN_NUMBER, RUN_URL, SHA (set in step env:)

'use strict';

const fs = require('fs');
const MARKER = '<!-- zynax-bdd-report -->';

module.exports = async function postBddComment({ github, context, core }) {
  const runNumber = process.env.RUN_NUMBER || '?';
  const runUrl    = process.env.RUN_URL    || '';
  const sha       = (process.env.SHA       || '').slice(0, 7);

  const resultsFile = 'protos/tests/bdd-results.txt';
  let body;

  if (fs.existsSync(resultsFile)) {
    const lines = fs.readFileSync(resultsFile, 'utf8').split('\n');
    const pkgResults = [];
    let passed = 0, failed = 0;

    for (const line of lines) {
      const okMatch   = line.match(/^ok\s+\S+\/protos\/tests\/(\S+)\s+(\S+)/);
      const failMatch = line.match(/^FAIL\s+\S+\/protos\/tests\/(\S+)/);
      if (line.match(/\s+--- PASS: Test\w+\/.+/)) passed++;
      if (line.match(/\s+--- FAIL: Test\w+\/.+/)) failed++;
      if (okMatch)   pkgResults.push({ pkg: okMatch[1],   status: '✅', time: okMatch[2] });
      if (failMatch) pkgResults.push({ pkg: failMatch[1], status: '❌', time: '—' });
    }

    const total    = passed + failed;
    const headline = failed === 0
      ? `✅ **${total} scenarios passed** across ${pkgResults.length} package(s)`
      : `❌ **${failed} scenario(s) failed** (${passed} passed) across ${pkgResults.length} package(s)`;
    const rows = pkgResults.length > 0
      ? pkgResults.map(r => `| \`${r.pkg}\` | ${r.status} | ${r.time} |`).join('\n')
      : '| — | ⏭️ skipped | — |';

    body = `${MARKER}\n## Contract Test Report\n\n${headline}\n\n| Package | Result | Time |\n|---------|--------|------|\n${rows}\n\n<sub>Run [#${runNumber}](${runUrl}) · \`${sha}\`</sub>`;
  } else {
    body = `${MARKER}\n## Contract Test Report\n\nℹ️ BDD suite skipped — no proto or contract test files changed in this PR.\n\n<sub>Run [#${runNumber}](${runUrl})</sub>`;
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
