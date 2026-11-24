// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

import type { GitProject, MachineSession, Project, Usage, Worker } from '../api/types';

export function git(over: Partial<GitProject>): GitProject {
  return { branch: 'main', head: 'abc1234', last_commit: '2026-11-09T09:00:00+01:00', remote: true, ahead: 0, dirty: 0, ...over };
}

export function usage(tokens: number): Usage {
  return { input: 0, output: tokens, cache_write: 0, cache_read: tokens * 10, messages: 1, tokens, cost: null };
}

export function project(over: Partial<Project>): Project {
  return {
    key: 'alpha/fern', tracked: true, state: 'active', group: '', parked_reason: '', superseded_by: '',
    updated: '2026-11-09', next: 'Run the review.', stopped: 'Task 17 paused.', git: {}, warnings: [], last_session: null,
    usage_7: usage(0), usage_30: usage(0), usage_all: usage(0),
    ...over,
  };
}

export function session(over: Partial<MachineSession>): MachineSession {
  return {
    machine: 'm1', id: 's1', project: 'alpha/fern', branch: 'main', title: 'Review task 17', remote: '',
    started: '2026-11-11T08:00:00+01:00', last: '2026-11-11T09:55:00+01:00', ended: false,
    ...over,
  };
}

export function worker(over: Partial<Worker>): Worker {
  return {
    name: 'm1', collected: '2026-11-11T10:00:00+01:00', up: true, claude_code: '2.1.323', behind: false,
    memories: null, sessions: [], checkouts: [],
    ...over,
  };
}
