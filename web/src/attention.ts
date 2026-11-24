// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

import type { GitProject, Project } from './api/types';
import { daysBetween } from './dates';
import { count } from './format';
import { byHandoff } from './project';

export type Kind = 'unpushed' | 'uncommitted' | 'stale';
export type Attention = { project: Project; kind: Kind; sentence: string };

const KINDS: Kind[] = ['unpushed', 'uncommitted', 'stale'];

// raises applies the rule the server uses to name a warning, to the git
// facts of one machine.
function raises(kind: Kind, g: GitProject, updated: string): boolean {
  if (kind === 'unpushed') return g.ahead > 0 || !g.remote;
  if (kind === 'uncommitted') return g.dirty > 0;
  return g.last_commit.slice(0, 10) > updated;
}

function words(kind: Kind, g: GitProject, updated: string): string {
  if (kind === 'uncommitted') return `${count(g.dirty, 'file is', 'files are')} uncommitted`;
  if (kind === 'stale') {
    const days = daysBetween(updated, g.last_commit.slice(0, 10));
    return `The status is older than the last commit, by ${count(days, 'day', 'days')}`;
  }
  if (!g.remote) {
    return g.ahead > 0 ? `No remote: ${count(g.ahead, 'commit exists', 'commits exist')} on this machine only` : 'No remote';
  }
  const pushed = `${count(g.ahead, 'commit is', 'commits are')} not pushed`;
  return g.branch === '' ? pushed : `${pushed}, on \`${g.branch}\``;
}

// sentence states one warning of a project, with the figures of the
// first machine, in name order, whose git facts raise it. It is null
// when no machine does.
export function sentence(p: Project, kind: Kind): string | null {
  const machines = Object.keys(p.git).sort();
  const machine = machines.find((m) => raises(kind, p.git[m]!, p.updated));
  if (machine === undefined) return null;
  const text = words(kind, p.git[machine]!, p.updated);
  return machines.length > 1 ? `${text}, seen on ${machine}` : text;
}

// attention lists every warning of every project, projects by most
// recent handoff.
export function attention(projects: Project[]): Attention[] {
  const out: Attention[] = [];
  for (const project of byHandoff(projects)) {
    for (const kind of KINDS) {
      if (!project.warnings.includes(kind)) continue;
      const text = sentence(project, kind);
      if (text !== null) out.push({ project, kind, sentence: text });
    }
  }
  return out;
}
