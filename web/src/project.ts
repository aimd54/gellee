// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

import type { Project } from './api/types';

// nameOf is the part of a key after the root.
export function nameOf(key: string): string {
  const i = key.indexOf('/');
  return i < 0 ? key : key.slice(i + 1);
}

export function rootOf(key: string): string {
  const i = key.indexOf('/');
  return i < 0 ? '' : key.slice(0, i);
}

// projectPath is the address of a project's page, and of its API
// resource: each segment of the key is encoded, so a key may hold
// characters such as # or %.
export function projectPath(key: string): string {
  return `/projects/${key.split('/').map(encodeURIComponent).join('/')}`;
}

// groupId is the id of a group's heading on the Projects view. It is a
// single token with no space, so it can name the section in
// aria-labelledby, and it is the fragment the breadcrumb links to.
export function groupId(group: string): string {
  return `group-${encodeURIComponent(group)}`;
}

// sharedNames returns the names that more than one of the keys carry.
export function sharedNames(keys: string[]): Set<string> {
  const seen = new Set<string>();
  const shared = new Set<string>();
  for (const key of new Set(keys)) {
    const name = nameOf(key);
    if (seen.has(name)) shared.add(name);
    seen.add(name);
  }
  return shared;
}

// label is the name of a project, with its root where the name alone
// would be ambiguous.
export function label(key: string, shared: Set<string>): string {
  const name = nameOf(key);
  const root = rootOf(key);
  return shared.has(name) && root !== '' ? `${name} (${root})` : name;
}

export function byHandoff(projects: Project[]): Project[] {
  return [...projects].sort((a, b) => b.updated.localeCompare(a.updated) || a.key.localeCompare(b.key));
}

export function byUsage(projects: Project[]): Project[] {
  return [...projects].sort((a, b) => b.usage_30.tokens - a.usage_30.tokens || a.key.localeCompare(b.key));
}

// stateLine is the one line shown under a project's name.
export function stateLine(p: Project, shared: Set<string>): string {
  if (p.state === 'parked') return `Parked: ${p.parked_reason || 'no reason given'}`;
  if (p.state === 'done' && p.superseded_by) return `Replaced by ${label(p.superseded_by, shared)}`;
  return p.next || 'No next action.';
}
