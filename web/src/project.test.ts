// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

import { expect, test } from 'vitest';
import { byHandoff, byUsage, groupId, label, nameOf, projectPath, rootOf, sharedNames, stateLine } from './project';
import { project, usage } from './test/shapes';

test('a project is named by the part of its key after the root', () => {
  expect(nameOf('alpha/fern')).toBe('fern');
  expect(rootOf('alpha/fern')).toBe('alpha');
  expect(nameOf('other')).toBe('other');
  expect(rootOf('other')).toBe('');
});

test('the root is added where two projects share a name', () => {
  const shared = sharedNames(['alpha/tool', 'beta/tool', 'alpha/fern', 'alpha/fern']);
  expect([...shared]).toEqual(['tool']);
  expect(label('alpha/tool', shared)).toBe('tool (alpha)');
  expect(label('beta/tool', shared)).toBe('tool (beta)');
  expect(label('alpha/fern', shared)).toBe('fern');
});

const projects = [
  project({ key: 'alpha/fern', updated: '2026-11-09', usage_30: usage(3330) }),
  project({ key: 'beta/tool', updated: '2026-11-11', usage_30: usage(555) }),
  project({ key: 'alpha/docs', updated: '2026-11-09', usage_30: usage(9000) }),
];

test('projects are ordered by most recent handoff, then by key', () => {
  expect(byHandoff(projects).map((p) => p.key)).toEqual(['beta/tool', 'alpha/docs', 'alpha/fern']);
  expect(projects[0]!.key).toBe('alpha/fern');
});

test('projects are ordered by usage over 30 days', () => {
  expect(byUsage(projects).map((p) => p.key)).toEqual(['alpha/docs', 'alpha/fern', 'beta/tool']);
});

test('the line of a project depends on its state', () => {
  const none = new Set<string>();
  expect(stateLine(project({ next: 'Run the review.' }), none)).toBe('Run the review.');
  expect(stateLine(project({ next: '' }), none)).toBe('No next action.');
  expect(stateLine(project({ state: 'parked', parked_reason: 'Waiting' }), none)).toBe('Parked: Waiting');
  expect(stateLine(project({ state: 'parked' }), none)).toBe('Parked: no reason given');
  expect(stateLine(project({ state: 'done', superseded_by: 'alpha/fern' }), none)).toBe('Replaced by fern');
  expect(stateLine(project({ state: 'done', next: 'Nothing.' }), none)).toBe('Nothing.');
});

test('a project path encodes each segment and keeps the slash between them', () => {
  expect(projectPath('alpha/fern')).toBe('/projects/alpha/fern');
  expect(projectPath('alpha/c#')).toBe('/projects/alpha/c%23');
  expect(projectPath('alpha/100%')).toBe('/projects/alpha/100%25');
  expect(projectPath('my root/my tool')).toBe('/projects/my%20root/my%20tool');
  expect(projectPath('alpha/a?b&c')).toBe('/projects/alpha/a%3Fb%26c');
});

test('a group id is one token that holds no space', () => {
  expect(groupId('tools')).toBe('group-tools');
  expect(groupId('data fern')).toBe('group-data%20fern');
  expect(groupId('a&b?c')).toBe('group-a%26b%3Fc');
});
