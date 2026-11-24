// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

export type Usage = {
  input: number;
  output: number;
  cache_write: number;
  cache_read: number;
  messages: number;
  tokens: number;
  cost: number | null;
};

export type GitProject = {
  branch: string;
  head: string;
  last_commit: string;
  remote: boolean;
  ahead: number;
  dirty: number;
};

export type Sections = { stopped: string; next: string; remaining: string; ideas: string };

export type Project = {
  key: string;
  tracked: boolean;
  state: string;
  group: string;
  parked_reason: string;
  superseded_by: string;
  updated: string;
  next: string;
  stopped: string;
  sections?: Sections;
  git: Record<string, GitProject>;
  warnings: string[];
  last_session: MachineSession | null;
  usage_7: Usage;
  usage_30: Usage;
  usage_all: Usage;
};

export type Problem = { file: string; message: string };
export type DayTokens = { day: string; tokens: Record<string, number> };

export type OverviewData = {
  built: string;
  today: string;
  limit: number;
  active: number;
  over: number;
  projects: Project[];
  days: DayTokens[];
  problems: Problem[];
  collected: string;
  week: Usage;
};

export type ProjectsData = { projects: Project[] };

export type DetailData = {
  project: Project;
  group_members: string[];
  days: { day: string; model: string; usage: Usage }[];
  sessions: MachineSession[];
};

export type UsageData = {
  from: string;
  to: string;
  total: Usage;
  projects: { key: string; tracked: boolean; usage: Usage }[];
  models: { model: string; usage: Usage }[];
  days: { day: string; usage: Usage }[];
};

export type IdeasData = { ideas: { project: string; text: string }[] };

export type HealthData = { today: string };

export type Session = {
  id: string;
  project: string;
  branch: string;
  title: string;
  remote: string;
  started: string;
  last: string;
  ended: boolean;
};

export type MachineSession = Session & { machine: string };

export type Checkout = { project: string; branch: string; warnings: string[] };

export type MemorySync = { pushed: string; set_aside: number; stale: boolean };

export type Worker = {
  name: string;
  collected: string;
  up: boolean;
  claude_code: string;
  behind: boolean;
  memories: MemorySync | null;
  sessions: (Session & { tracked: boolean })[];
  checkouts: Checkout[];
};

export type WorkersData = { built: string; today: string; workers: Worker[]; problems: Problem[] };
