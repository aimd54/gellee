// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

import { fireEvent, render, screen, within } from '@testing-library/react';
import { expect, test, vi } from 'vitest';
import type { Project } from '../api/types';
import { stubApi } from '../test/api';
import { git, project, session, usage } from '../test/shapes';
import { Overview } from './Overview';

const projects = [
  project({
    key: 'alpha/fern', group: 'fieldwork', updated: '2026-11-09',
    stopped: 'Task 17 paused. Branch `m2-task-17`.', next: 'Run the review.',
    warnings: ['stale', 'unpushed'], usage_7: usage(3330),
    git: { m1: git({ branch: 'm2-task-17', ahead: 2, last_commit: '2026-11-10T09:00:00+01:00' }) },
  }),
  project({ key: 'alpha/fern-docs', state: 'parked', group: 'fieldwork', parked_reason: 'Waiting for milestone 2', updated: '2026-10-20' }),
  project({ key: 'alpha/renamed', updated: '2026-11-10', stopped: 'The rename is done.', next: 'Update the links.', usage_7: usage(222) }),
  project({ key: 'beta/legacy', state: 'done', superseded_by: 'alpha/fern', updated: '2026-09-01' }),
  project({
    key: 'beta/tool', updated: '2026-11-11',
    stopped: 'The parser handles nested lists.', next: 'Add the table parser.',
    warnings: ['uncommitted'], usage_7: usage(555),
    git: { m1: git({ dirty: 3, last_commit: '2026-11-11T08:00:00+01:00' }) },
  }),
];

const overview = {
  built: '2026-11-11T12:00:00Z',
  today: '2026-11-11',
  limit: 2,
  active: 3,
  over: 1,
  projects: [],
  days: [
    { day: '2026-11-10', tokens: { 'alpha/fern': 1110 } },
    { day: '2026-11-11', tokens: { 'alpha/fern': 2220, other: 777 } },
  ],
  problems: [{ file: 'alpha/broken.md', message: 'state "sleeping" is not active, parked or done' }],
  collected: '2026-11-11T10:00:00+01:00',
  week: usage(4884),
};

const ideas = {
  ideas: [
    { project: 'alpha/fern', text: 'Compare two mixes' },
    { project: 'alpha/fern', text: 'Write up the measurements' },
    { project: 'alpha/renamed', text: 'A note on renaming' },
    { project: 'beta/tool', text: 'A fourth idea' },
  ],
};

const month = { from: '2026-10-13', to: '2026-11-11', total: usage(4887), projects: [], models: [], days: [] };

function stub(over: { overview?: object; projects?: Project[]; ideas?: object } = {}) {
  stubApi({
    '/api/v1/overview': { ...overview, ...over.overview },
    '/api/v1/projects': { projects: over.projects ?? projects },
    '/api/v1/ideas': over.ideas ?? ideas,
    '/api/v1/usage': month,
  });
}

const stat = (label: string) => screen.getByText(label).parentElement!.querySelector('b')!.textContent;
const workbench = () => screen.getByRole('region', { name: 'On your workbench' });
const rows = () => within(workbench()).getAllByRole('listitem');

test('the page head states the day, the collection time, the limit and what could not be read', async () => {
  stub();
  render(<Overview go={vi.fn()} />);
  expect(await screen.findByRole('heading', { level: 1 })).toHaveTextContent('Pick up the thread.');
  expect(screen.getByText('Wednesday, 11 November · collected 10:00')).toBeVisible();
  expect(screen.getByText('3 projects on the bench, the limit is 2.')).toBeVisible();
  expect(screen.getByText('1 over the limit')).toHaveClass('gl-tone-late');
  expect(screen.getByRole('alert')).toHaveTextContent('alpha/broken.md: state "sleeping" is not active, parked or done');
});

test('within the limit, nothing is said about going over', async () => {
  stub({ overview: { limit: 3, problems: [] } });
  render(<Overview go={vi.fn()} />);
  expect(await screen.findByText('3 projects on the bench, the limit is 3.')).toBeVisible();
  expect(screen.queryByText(/over the limit/)).toBeNull();
  expect(screen.queryByRole('alert')).toBeNull();
});

test('the bench holds the active projects, most recent handoff first, the first one tinted', async () => {
  stub();
  const go = vi.fn();
  render(<Overview go={go} />);
  const bench = await screen.findByRole('region', { name: 'The bench' });
  const cards = [...bench.querySelectorAll<HTMLElement>('a.gl-hero')];
  expect(cards.map((c) => c.getAttribute('data-project'))).toEqual(['beta/tool', 'alpha/renamed', 'alpha/fern']);

  expect(cards[0]).toHaveClass('gl-hero-first');
  expect(within(cards[0]!).getByText('Where you left off')).toBeVisible();
  expect(within(cards[0]!).getByRole('heading', { level: 2 })).toHaveTextContent('tool');
  expect(cards[0]).toHaveTextContent('The parser handles nested lists.');
  expect(cards[0]).toHaveTextContent('Add the table parser.');
  expect(within(cards[0]!).getByText('today')).toBeVisible();
  expect(within(cards[0]!).getByText('uncommitted work')).toBeVisible();
  expect(cards[0]).toHaveTextContent('555 in 7 days');

  expect(cards[1]).not.toHaveClass('gl-hero-first');
  expect(cards[1]).not.toHaveTextContent('Where you left off');
  expect(cards[1]).not.toHaveTextContent('The rename is done.');
  expect(within(cards[1]!).getByText('alpha')).toBeVisible();
  expect(within(cards[1]!).getByText('yesterday')).toBeVisible();
  expect(cards[1]).toHaveTextContent('Update the links.');

  expect(within(cards[2]!).getByText('fieldwork group')).toBeVisible();
  expect(within(cards[2]!).getByText('9 November')).toBeVisible();
  expect(within(cards[2]!).getByText('stale status')).toBeVisible();
  expect(cards[2]).toHaveTextContent('3.3k in 7 days');

  fireEvent.click(cards[2]!);
  expect(go).toHaveBeenCalledWith('/projects/alpha/fern');
});

test('an empty bench says how to fill it', async () => {
  stub({ projects: projects.filter((p) => p.state !== 'active') });
  render(<Overview go={vi.fn()} />);
  expect(await screen.findByText('No project on the bench, the limit is 2.')).toBeVisible();
  expect(screen.getByText(/Nothing on the bench/)).toHaveTextContent(
    'Nothing on the bench. Activate a project with /handoff activate.',
  );
  expect(screen.getByText('/handoff activate').tagName).toBe('CODE');
  expect(screen.queryByRole('region', { name: 'The bench' })).toBeNull();
});

test('the stat row counts projects, parked and done, attention and the week', async () => {
  stub();
  render(<Overview go={vi.fn()} />);
  await screen.findByRole('heading', { level: 1 });
  expect(stat('projects tracked')).toBe('5');
  expect(stat('parked, 1 done')).toBe('1');
  expect(stat('need attention')).toBe('2');
  expect(screen.getByText('need attention').parentElement!.querySelector('b')).toHaveClass('gl-late');
  expect(stat('tokens in 7 days')).toBe('4.9k');
});

test('the workbench opens on what needs attention, one row per project and warning', async () => {
  stub();
  render(<Overview go={vi.fn()} />);
  await screen.findByRole('heading', { level: 1 });
  expect(within(workbench()).getByRole('button', { name: 'Needs attention 3' })).toHaveAttribute('aria-pressed', 'true');
  expect(rows()).toHaveLength(3);
  expect(rows()[0]).toHaveTextContent('tool');
  expect(rows()[0]).toHaveTextContent('3 files are uncommitted');
  expect(rows()[1]).toHaveTextContent('fern');
  expect(rows()[1]).toHaveTextContent('2 commits are not pushed, on m2-task-17');
  expect(within(rows()[1]!).getByText('m2-task-17').tagName).toBe('CODE');
  expect(rows()[2]).toHaveTextContent('The status is older than the last commit, by 1 day');
  expect(within(rows()[2]!).getByText('active')).toBeVisible();
  expect(within(rows()[2]!).getByText('9 November')).toBeVisible();

  fireEvent.click(within(workbench()).getByRole('button', { name: 'Parked 1' }));
  expect(rows()).toHaveLength(1);
  expect(rows()[0]).toHaveTextContent('fern-docs');
  expect(rows()[0]).toHaveTextContent('Parked: Waiting for milestone 2');

  fireEvent.click(within(workbench()).getByRole('button', { name: 'All 5' }));
  expect(
    within(workbench())
      .getAllByRole('link')
      .map((a) => a.getAttribute('data-project'))
      .filter(Boolean),
  ).toEqual(['beta/tool', 'alpha/renamed', 'alpha/fern', 'alpha/fern-docs', 'beta/legacy']);
  expect(rows()[0]).toHaveTextContent('Add the table parser.');
});

test('when nothing needs attention the workbench opens on the parked projects', async () => {
  stub({ projects: projects.map((p) => ({ ...p, warnings: [] })) });
  render(<Overview go={vi.fn()} />);
  await screen.findByRole('heading', { level: 1 });
  expect(within(workbench()).getByRole('button', { name: 'Parked 1' })).toHaveAttribute('aria-pressed', 'true');
  expect(rows()[0]).toHaveTextContent('Parked: Waiting for milestone 2');
  expect(stat('need attention')).toBe('0');
  expect(screen.getByText('need attention').parentElement!.querySelector('b')).not.toHaveClass('gl-late');
  fireEvent.click(within(workbench()).getByRole('button', { name: 'Needs attention 0' }));
  expect(within(workbench()).getByText('Nothing needs attention.')).toBeVisible();
});

test('the side cards show the last 30 days and the first three ideas', async () => {
  stub();
  const go = vi.fn();
  render(<Overview go={go} />);
  const card = await screen.findByRole('region', { name: 'Last 30 days' });
  const chart = within(card).getByRole('img', { name: 'Tokens per day, by project' });
  expect(chart.querySelectorAll('g[data-day]')).toHaveLength(2);
  expect(chart.querySelectorAll('g[data-day="2026-11-11"] rect')).toHaveLength(2);
  expect(within(card).getByText('fern')).toBeVisible();
  expect(within(card).getByText('yesterday to today, at most 3k in a day')).toBeVisible();
  expect(within(card).getByText('4.9k tokens, 48.9k cache reads')).toBeVisible();
  fireEvent.click(within(card).getByRole('link', { name: 'Usage in detail' }));
  expect(go).toHaveBeenCalledWith('/usage');

  const later = screen.getByRole('region', { name: 'Set aside for later' });
  expect(within(later).getAllByRole('listitem').map((li) => li.textContent)).toEqual([
    'Compare two mixesfern',
    'Write up the measurementsfern',
    'A note on renamingrenamed',
  ]);
  fireEvent.click(within(later).getByRole('link', { name: 'All ideas' }));
  expect(go).toHaveBeenCalledWith('/ideas');
});

test('without any idea the card for later is left out', async () => {
  stub({ ideas: { ideas: [] } });
  render(<Overview go={vi.fn()} />);
  await screen.findByRole('heading', { level: 1 });
  expect(screen.queryByRole('region', { name: 'Set aside for later' })).toBeNull();
});

test('the bench cards share one unit when only some of them are priced', async () => {
  const mixed = projects.map((p) => (p.key === 'beta/tool' ? { ...p, usage_7: { ...p.usage_7, cost: 1.5 } } : p));
  stub({ projects: mixed });
  render(<Overview go={vi.fn()} />);
  const bench = await screen.findByRole('region', { name: 'The bench' });
  const cards = [...bench.querySelectorAll<HTMLElement>('a.gl-hero')];
  expect(cards[0]).toHaveTextContent('555 in 7 days');
  expect(cards[2]).toHaveTextContent('3.3k in 7 days');
  for (const c of cards) expect(c).not.toHaveTextContent('$');
});

test('a bench card carries the machine and time of the last session, and Open', async () => {
  const withSession = projects.map((p) =>
    p.key === 'beta/tool'
      ? { ...p, last_session: session({ machine: 'm1', project: 'beta/tool', remote: 'session_01Tool', last: '2026-11-11T09:55:00+01:00' }) }
      : p,
  );
  stub({ projects: withSession });
  render(<Overview go={vi.fn()} />);
  const bench = await screen.findByRole('region', { name: 'The bench' });
  const wraps = bench.querySelectorAll('.gl-hero-wrap');
  expect(wraps).toHaveLength(3);
  expect(wraps[0]).toHaveTextContent('Last session on m1, today, 09:55');
  expect(within(wraps[0] as HTMLElement).getByRole('link', { name: /^Open\b/ })).toHaveAttribute('href', 'https://claude.ai/code/session_01Tool');
  expect(wraps[0]!.querySelector('a.gl-hero a')).toBeNull();
  expect(wraps[1]).not.toHaveTextContent('Last session');
});
