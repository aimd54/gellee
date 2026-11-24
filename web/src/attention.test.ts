// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

import { expect, test } from 'vitest';
import { attention, sentence } from './attention';
import { git, project } from './test/shapes';

const on = (g: Parameters<typeof git>[0], over: Parameters<typeof project>[0] = {}) =>
  project({ updated: '2026-11-09', git: { m1: git(g) }, ...over });

test('unpushed commits name their branch', () => {
  expect(sentence(on({ ahead: 67 }), 'unpushed')).toBe('67 commits are not pushed, on `main`');
  expect(sentence(on({ ahead: 1, branch: 'fix' }), 'unpushed')).toBe('1 commit is not pushed, on `fix`');
});

test('without a remote the commits exist on one machine only', () => {
  expect(sentence(on({ remote: false, ahead: 20 }), 'unpushed')).toBe('No remote: 20 commits exist on this machine only');
  expect(sentence(on({ remote: false, ahead: 1 }), 'unpushed')).toBe('No remote: 1 commit exists on this machine only');
});

test('uncommitted files are counted', () => {
  expect(sentence(on({ dirty: 3 }), 'uncommitted')).toBe('3 files are uncommitted');
  expect(sentence(on({ dirty: 1 }), 'uncommitted')).toBe('1 file is uncommitted');
});

test('a stale status says by how many days', () => {
  const late = { last_commit: '2026-11-10T18:00:00+01:00' };
  expect(sentence(on(late, { updated: '2026-10-29' }), 'stale')).toBe('The status is older than the last commit, by 12 days');
  expect(sentence(on(late), 'stale')).toBe('The status is older than the last commit, by 1 day');
});

test('a warning no machine raises has no sentence', () => {
  expect(sentence(on({}), 'unpushed')).toBeNull();
  expect(sentence(on({}), 'uncommitted')).toBeNull();
  expect(sentence(on({}), 'stale')).toBeNull();
  expect(sentence(project({ git: {} }), 'unpushed')).toBeNull();
});

test('with several machines, the first in name order that raises the warning is named', () => {
  const p = project({ git: { m3: git({ dirty: 9 }), m2: git({ dirty: 3 }), m1: git({ dirty: 0 }) } });
  expect(sentence(p, 'uncommitted')).toBe('3 files are uncommitted, seen on m2');
});

test('attention lists one row per project and warning, most recent handoff first', () => {
  const rows = attention([
    project({
      key: 'alpha/fern', updated: '2026-11-09', warnings: ['stale', 'unpushed'],
      git: { m1: git({ branch: 'm2-task-17', ahead: 2, last_commit: '2026-11-10T09:00:00+01:00' }) },
    }),
    project({ key: 'alpha/docs', state: 'parked', updated: '2026-11-10' }),
    project({ key: 'beta/tool', updated: '2026-11-11', warnings: ['uncommitted'], git: { m1: git({ dirty: 3 }) } }),
  ]);
  expect(rows.map((r) => `${r.project.key} ${r.kind}: ${r.sentence}`)).toEqual([
    'beta/tool uncommitted: 3 files are uncommitted',
    'alpha/fern unpushed: 2 commits are not pushed, on `m2-task-17`',
    'alpha/fern stale: The status is older than the last commit, by 1 day',
  ]);
});

test('a detached head has no branch to name', () => {
  expect(sentence(on({ ahead: 3, branch: '' }), 'unpushed')).toBe('3 commits are not pushed');
  expect(sentence(on({ ahead: 1, branch: '' }), 'unpushed')).toBe('1 commit is not pushed');
});

test('a repository without commits has nothing to count', () => {
  const empty = { branch: 'main', head: '', last_commit: '', remote: false, ahead: 0 };
  expect(sentence(on(empty), 'unpushed')).toBe('No remote');
  const p = on(empty, { warnings: ['unpushed'] });
  expect(attention([p]).map((r) => r.sentence)).toEqual(['No remote']);
  expect(sentence(on(empty), 'stale')).toBeNull();
});
