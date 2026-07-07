// SPDX-License-Identifier: Apache-2.0
// Build the BDD contract-test report comment body from the Go test output.
//
// Pure builder: reads protos/tests/bdd-results.txt + RUN_* env and returns the
// markdown body. It performs NO GitHub API calls — the untrusted, PR-triggered
// test-go job runs it to produce an artifact, and the trusted pr-comment.yml
// workflow (workflow_run) posts that body. Keeping the write path out of the
// fork-controlled job is what lets fork PRs get the comment without a
// write-scoped token ever reaching attacker-controllable context (#1668).

'use strict';

const fs = require('fs');

const MARKER = '<!-- zynax-bdd-report -->';

function buildBddComment() {
  const runNumber = process.env.RUN_NUMBER || '?';
  const runUrl    = process.env.RUN_URL    || '';
  const sha       = (process.env.SHA       || '').slice(0, 7);

  const resultsFile = 'protos/tests/bdd-results.txt';
  if (!fs.existsSync(resultsFile)) {
    return `${MARKER}\n## Contract Test Report\n\nℹ️ BDD suite skipped — no proto or contract test files changed in this PR.\n\n<sub>Run [#${runNumber}](${runUrl})</sub>`;
  }

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

  return `${MARKER}\n## Contract Test Report\n\n${headline}\n\n| Package | Result | Time |\n|---------|--------|------|\n${rows}\n\n<sub>Run [#${runNumber}](${runUrl}) · \`${sha}\`</sub>`;
}

module.exports = { buildBddComment, MARKER };
