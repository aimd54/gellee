// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

import { cleanup, fireEvent, render, screen, within } from '@testing-library/react';
import { expect, test, vi } from 'vitest';
import { stubApi } from '../test/api';
import { git, project, session, usage } from '../test/shapes';
import { dominant, ProjectPage } from './ProjectPage';

const health = { today: '2026-11-11' };

const fern = {
  project: {
    ...project({
      key: 'alpha/fern', group: 'fieldwork', updated: '2026-11-09', warnings: ['stale', 'unpushed'],
      usage_7: usage(3330), usage_30: usage(3330), usage_all: usage(14430),
      git: { m1: git({ branch: 'm2-task-17', head: 'abc1234', last_commit: '2026-11-10T09:00:00+01:00', ahead: 2 }) },
    }),
    sections: { stopped: 'Task 17 paused. Branch `m2-task-17`.', next: 'Run the review.', remaining: '- Tasks 18 to 24', ideas: '' },
  },
  group_members: ['alpha/fern-docs'],
  days: [
    { day: '2026-11-11', model: 'model-a', usage: usage(2220) },
    { day: '2026-11-10', model: 'model-b', usage: usage(1110) },
    { day: '2026-10-01', model: 'model-a', usage: usage(11100) },
  ],
  sessions: [],
};

const docs = {
  project: {
    ...project({ key: 'alpha/fern-docs', state: 'parked', group: 'fieldwork', parked_reason: 'Waiting for milestone 2', updated: '2026-10-20' }),
    sections: { stopped: '', next: 'Review the install page.', remaining: '', ideas: '' },
  },
  group_members: ['alpha/fern'],
  days: [],
  sessions: [],
};

const legacy = {
  project: {
    ...project({ key: 'beta/legacy', state: 'done', superseded_by: 'alpha/fern', updated: '2026-09-01' }),
    sections: { stopped: 'Replaced.', next: '', remaining: '', ideas: '- Keep the notes' },
  },
  group_members: [],
  days: [],
  sessions: [],
};

const region = (name: string) => screen.getByRole('region', { name });

test('the head names the project, its state, its root, its handoff and its group', async () => {
  stubApi({ '/api/v1/projects/alpha/fern': fern, '/api/v1/health': health });
  const go = vi.fn();
  render(<ProjectPage projectKey="alpha/fern" go={go} />);
  expect(await screen.findByRole('heading', { level: 1 })).toHaveTextContent('fern');

  const crumb = screen.getByRole('navigation', { name: 'Breadcrumb' });
  expect(within(crumb).getAllByRole('link').map((a) => [a.textContent, a.getAttribute('href')])).toEqual([
    ['Projects', '/projects'],
    ['fieldwork', '/projects#group-fieldwork'],
  ]);

  const head = screen.getByRole('banner');
  expect(within(head).getByText('active')).toBeVisible();
  expect(within(head).getByText('stale status')).toBeVisible();
  expect(within(head).getByText('unpushed work')).toBeVisible();
  expect(within(head).getByText('alpha')).toBeVisible();
  expect(within(head).getByText('last handoff 9 November')).toBeVisible();
  const member = within(head).getByRole('link', { name: 'fern-docs' });
  expect(member).toHaveAttribute('href', '/projects/alpha/fern-docs');
  fireEvent.click(member);
  expect(go).toHaveBeenCalledWith('/projects/alpha/fern-docs');
});

test('the wide column reads next action, where I stopped, remaining and what is set aside', async () => {
  stubApi({ '/api/v1/projects/alpha/fern': fern, '/api/v1/health': health });
  render(<ProjectPage projectKey="alpha/fern" go={vi.fn()} />);
  await screen.findByRole('heading', { level: 1 });
  expect(region('Next action')).toHaveTextContent('Run the review.');
  expect(region('Next action')).not.toHaveTextContent('Parked');
  expect(within(region('Where I stopped')).getByText('m2-task-17').tagName).toBe('CODE');
  expect(within(region('Remaining')).getByRole('listitem')).toHaveTextContent('Tasks 18 to 24');
  expect(within(region('Set aside for later')).getByText('No idea recorded yet.')).toBeVisible();
});

test('a git card states the branch, the last commit and what is not pushed or committed', async () => {
  stubApi({ '/api/v1/projects/alpha/fern': fern, '/api/v1/health': health });
  render(<ProjectPage projectKey="alpha/fern" go={vi.fn()} />);
  const card = await screen.findByRole('region', { name: 'Git, m1' });
  expect(within(card).getByText('m2-task-17')).toHaveClass('gl-mono');
  const commit = within(card).getByText(/yesterday, 09:00/);
  expect(commit).toHaveTextContent('yesterday, 09:00 abc1234');
  expect(commit).toHaveClass('gl-late');
  expect(within(card).getByText('2 commits')).toHaveClass('gl-late');
  expect(within(card).getByText('nothing')).not.toHaveClass('gl-late');
});

test('the usage card gives three figures, the bars, the dominant model and a folded table', async () => {
  stubApi({ '/api/v1/projects/alpha/fern': fern, '/api/v1/health': health });
  render(<ProjectPage projectKey="alpha/fern" go={vi.fn()} />);
  const card = await screen.findByRole('region', { name: 'Usage' });
  expect(within(card).getByText('7 days').parentElement).toHaveTextContent('3.3k');
  expect(within(card).getByText('30 days').parentElement).toHaveTextContent('3.3k');
  expect(within(card).getByText('in total').parentElement).toHaveTextContent('14.4k');

  const chart = within(card).getByRole('img', { name: 'Tokens per day, last 30 days' });
  expect(chart.querySelectorAll('g[data-day]')).toHaveLength(30);
  expect(chart.querySelectorAll('g[data-day="2026-11-11"] rect')).toHaveLength(1);
  expect(chart.querySelectorAll('g[data-day="2026-11-09"] rect')).toHaveLength(0);
  expect(chart.querySelector('g[data-day="2026-10-01"]')).toBeNull();
  expect(within(card).getByText('144k cache reads in total, mostly model-a')).toBeVisible();

  const fold = within(card).getByRole('button', { name: 'Day by day, per model (3 rows)' });
  expect(fold).toHaveAttribute('aria-expanded', 'false');
  expect(within(card).queryByRole('table')).toBeNull();
  fireEvent.click(fold);
  expect(fold).toHaveAttribute('aria-expanded', 'true');
  // The table has a name, the toggle points at what it opens, and the
  // wrapper that scrolls sideways can be reached by keyboard.
  const table = within(card).getByRole('table', { name: 'Day by day, per model' });
  const scroller = within(card).getByRole('region', { name: 'Day by day, per model' });
  expect(scroller).toHaveAttribute('tabindex', '0');
  expect(scroller).toContainElement(table);
  expect(fold.getAttribute('aria-controls')).toBe(scroller.id);
  expect(scroller.id).not.toBe('');
  for (const td of within(card).getAllByText('model-a', { selector: 'td' })) expect(td).not.toHaveClass('gl-mono');
  const rows = within(card).getAllByRole('row');
  expect(rows).toHaveLength(4);
  expect(rows[1]).toHaveTextContent('today');
  expect(rows[1]).toHaveTextContent('model-a');
  expect(rows[1]).toHaveTextContent('2.2k');
  expect(rows[3]).toHaveTextContent('1 October');
  expect(rows[3]).toHaveTextContent('11.1k');
});

test('a parked project says why above its next action, and empty sections say so', async () => {
  stubApi({ '/api/v1/projects/alpha/fern-docs': docs, '/api/v1/health': health });
  render(<ProjectPage projectKey="alpha/fern-docs" go={vi.fn()} />);
  await screen.findByRole('heading', { level: 1 });
  const next = region('Next action');
  expect(next).toHaveTextContent('Parked: Waiting for milestone 2');
  expect(next).toHaveTextContent('Review the install page.');
  expect(next.textContent!.indexOf('Parked')).toBeLessThan(next.textContent!.indexOf('Review'));
  expect(within(region('Where I stopped')).getByText('No handoff yet.')).toBeVisible();
  expect(within(region('Remaining')).getByText('Nothing listed.')).toBeVisible();
  expect(within(region('Git')).getByText('No machine has reported this working tree.')).toBeVisible();
  expect(within(region('Usage')).getByText('No usage in this period.')).toBeVisible();
  expect(within(region('Usage')).queryByRole('button')).toBeNull();
});

test('a replaced project links to its replacement and has no group in its breadcrumb', async () => {
  stubApi({ '/api/v1/projects/beta/legacy': legacy, '/api/v1/health': health });
  render(<ProjectPage projectKey="beta/legacy" go={vi.fn()} />);
  await screen.findByRole('heading', { level: 1 });
  const head = screen.getByRole('banner');
  expect(within(head).getByText('done')).toBeVisible();
  expect(within(head).getByText('last handoff 1 September')).toBeVisible();
  expect(within(head).getByRole('link', { name: 'fern' })).toHaveAttribute('href', '/projects/alpha/fern');
  expect(within(screen.getByRole('navigation', { name: 'Breadcrumb' })).getAllByRole('link')).toHaveLength(1);
  expect(region('Next action')).toHaveTextContent('No next action.');
  expect(within(region('Set aside for later')).getByRole('listitem')).toHaveTextContent('Keep the notes');
});

test('an unknown project says so and links back to the list', async () => {
  stubApi({ '/api/v1/health': health });
  const go = vi.fn();
  render(<ProjectPage projectKey="alpha/nope" go={go} />);
  expect(await screen.findByRole('heading', { level: 1 })).toHaveTextContent('No project at this address');
  fireEvent.click(screen.getByRole('link', { name: 'Back to Projects' }));
  expect(go).toHaveBeenCalledWith('/projects');
});

test('the dominant model is the one with the most tokens, the first by name on a tie', () => {
  expect(dominant(fern.days)).toBe('model-a');
  expect(dominant([{ day: '2026-11-11', model: 'b', usage: usage(5) }, { day: '2026-11-10', model: 'a', usage: usage(5) }])).toBe('a');
  expect(dominant([])).toBeNull();
});

test('the breadcrumb link to a group whose name needs encoding points at its id', async () => {
  const p = { ...fern, project: { ...fern.project, group: 'a&b?c' } };
  stubApi({ '/api/v1/projects/alpha/fern': p, '/api/v1/health': health });
  render(<ProjectPage projectKey="alpha/fern" go={vi.fn()} />);
  await screen.findByRole('heading', { level: 1 });
  const crumb = screen.getByRole('navigation', { name: 'Breadcrumb' });
  expect(within(crumb).getByRole('link', { name: 'a&b?c' })).toHaveAttribute('href', '/projects#group-a%26b%3Fc');
});

function withGit(over: Parameters<typeof git>[0], updated = '2026-11-09') {
  return {
    project: { ...project({ key: 'alpha/odd', updated, git: { m1: git(over) } }), sections: { stopped: '', next: '', remaining: '', ideas: '' } },
    group_members: [],
    days: [],
    sessions: [],
  };
}

test('a git card says detached for a detached head', async () => {
  stubApi({ '/api/v1/projects/alpha/odd': withGit({ branch: '' }), '/api/v1/health': health });
  render(<ProjectPage projectKey="alpha/odd" go={vi.fn()} />);
  const card = await screen.findByRole('region', { name: 'Git, m1' });
  expect(within(card).getByText('detached')).toBeVisible();
  expect(within(card).getByText('detached')).not.toHaveClass('gl-mono');
});

test('a git card for a repository without commits says so', async () => {
  stubApi({
    '/api/v1/projects/alpha/odd': withGit({ branch: 'main', head: '', last_commit: '', remote: false, ahead: 0 }),
    '/api/v1/health': health,
  });
  render(<ProjectPage projectKey="alpha/odd" go={vi.fn()} />);
  const card = await screen.findByRole('region', { name: 'Git, m1' });
  const dd = (term: string) => within(card).getByText(term).nextElementSibling!;
  expect(dd('Last commit')).toHaveTextContent(/^no commits$/);
  expect(dd('Unpushed')).toHaveTextContent(/^no remote$/);
  expect(dd('Unpushed')).toHaveClass('gl-late');
});

test('links that are only muted text are underlined at rest', async () => {
  stubApi({ '/api/v1/projects/alpha/fern': fern, '/api/v1/health': health });
  render(<ProjectPage projectKey="alpha/fern" go={vi.fn()} />);
  await screen.findByRole('heading', { level: 1 });
  const crumb = screen.getByRole('navigation', { name: 'Breadcrumb' });
  for (const a of within(crumb).getAllByRole('link')) expect(a).toHaveClass('gl-quiet');
});

test('the folded table shows one unit in every row', async () => {
  const priced = { ...usage(1200), cost: 0.12 };
  const mixed = {
    ...fern,
    days: [
      { day: '2026-11-11', model: 'model-a', usage: priced },
      { day: '2026-11-10', model: 'model-b', usage: usage(1200) },
    ],
  };
  stubApi({ '/api/v1/projects/alpha/fern': mixed, '/api/v1/health': health });
  render(<ProjectPage projectKey="alpha/fern" go={vi.fn()} />);
  fireEvent.click(await screen.findByRole('button', { name: /^Day by day, per model/ }));
  const rows = within(screen.getByRole('table', { name: 'Day by day, per model' })).getAllByRole('row');
  expect(rows[1]).toHaveTextContent('1.2k');
  expect(rows[2]).toHaveTextContent('1.2k');
  expect(rows[1]).not.toHaveTextContent('$');

  const allPriced = { ...mixed, days: mixed.days.map((d) => ({ ...d, usage: priced })) };
  stubApi({ '/api/v1/projects/alpha/fern': allPriced, '/api/v1/health': health });
  cleanup();
  render(<ProjectPage projectKey="alpha/fern" go={vi.fn()} />);
  fireEvent.click(await screen.findByRole('button', { name: /^Day by day, per model/ }));
  expect(within(screen.getByRole('table')).getAllByRole('row')[1]).toHaveTextContent('$0.12');
});

test('the three usage figures share one unit', async () => {
  const p = fern.project;
  const recent = { ...fern, project: { ...p, usage_7: { ...p.usage_7, cost: 0.4 }, usage_30: { ...p.usage_30, cost: 0.4 } } };
  stubApi({ '/api/v1/projects/alpha/fern': recent, '/api/v1/health': health });
  render(<ProjectPage projectKey="alpha/fern" go={vi.fn()} />);
  const card = await screen.findByRole('region', { name: 'Usage' });
  expect(within(card).getByText('7 days').parentElement).toHaveTextContent('3.3k');
  expect(within(card).getByText('30 days').parentElement).toHaveTextContent('3.3k');
  expect(within(card).getByText('in total').parentElement).toHaveTextContent('14.4k');

  const all = { ...recent, project: { ...recent.project, usage_all: { ...p.usage_all, cost: 1.9 } } };
  stubApi({ '/api/v1/projects/alpha/fern': all, '/api/v1/health': health });
  cleanup();
  render(<ProjectPage projectKey="alpha/fern" go={vi.fn()} />);
  const priced = await screen.findByRole('region', { name: 'Usage' });
  expect(within(priced).getByText('7 days').parentElement).toHaveTextContent('$0.40');
  expect(within(priced).getByText('in total').parentElement).toHaveTextContent('$1.90');
});

test('the Sessions card lists the project sessions on every worker', async () => {
  stubApi({
    '/api/v1/projects/alpha/fern': {
      ...fern,
      sessions: [
        session({ id: 'a', machine: 'm1', title: 'Review task 17', branch: 'm2-task-17', last: '2026-11-10T11:02:00+01:00' }),
        session({ id: 'b', machine: 'm2', title: 'Plan milestone 2', remote: 'session_01Fern', last: '2026-11-09T17:40:00+01:00' }),
      ],
    },
    '/api/v1/health': health,
  });
  render(<ProjectPage projectKey="alpha/fern" go={vi.fn()} />);
  const card = await screen.findByRole('region', { name: 'Sessions' });
  const rows = within(card).getAllByRole('listitem');
  expect(rows).toHaveLength(2);
  expect(rows[0]).toHaveTextContent('m1');
  expect(rows[0]).toHaveTextContent('yesterday, 11:02');
  expect(rows[1]).toHaveTextContent('m2');
  expect(within(rows[1]!).getByRole('link', { name: /^Open\b/ })).toHaveAttribute('href', 'https://claude.ai/code/session_01Fern');
});

test('the Sessions card folds ended sessions apart until asked for', async () => {
  stubApi({
    '/api/v1/projects/alpha/fern': {
      ...fern,
      sessions: [
        session({ id: 'a', machine: 'm1', title: 'Running one', last: '2026-11-10T11:02:00+01:00' }),
        session({ id: 'b', machine: 'm2', title: 'Done one', last: '2026-11-09T17:40:00+01:00', ended: true }),
        session({ id: 'c', machine: 'm2', title: 'Done two', last: '2026-11-09T16:40:00+01:00', ended: true }),
      ],
    },
    '/api/v1/health': health,
  });
  render(<ProjectPage projectKey="alpha/fern" go={vi.fn()} />);
  const card = await screen.findByRole('region', { name: 'Sessions' });
  expect(within(card).getAllByRole('listitem')).toHaveLength(1);
  expect(card).toHaveTextContent('Running one');
  expect(card).not.toHaveTextContent('Done one');
  expect(card).not.toHaveTextContent('No running session.');
  fireEvent.click(within(card).getByRole('button', { name: 'Ended (2)' }));
  expect(within(card).getAllByRole('listitem')).toHaveLength(3);
  expect(card).toHaveTextContent('Done two');
});

test('the Sessions card says no session runs when all ended', async () => {
  stubApi({
    '/api/v1/projects/alpha/fern': { ...fern, sessions: [session({ id: 'b', machine: 'm2', title: 'Done one', ended: true })] },
    '/api/v1/health': health,
  });
  render(<ProjectPage projectKey="alpha/fern" go={vi.fn()} />);
  const card = await screen.findByRole('region', { name: 'Sessions' });
  expect(card).toHaveTextContent('No running session.');
  expect(within(card).queryAllByRole('listitem')).toHaveLength(0);
  expect(within(card).getByRole('button', { name: 'Ended (1)' })).toBeVisible();
});

test('the Sessions card says when there is none', async () => {
  stubApi({ '/api/v1/projects/alpha/fern': fern, '/api/v1/health': health });
  render(<ProjectPage projectKey="alpha/fern" go={vi.fn()} />);
  expect(await screen.findByRole('region', { name: 'Sessions' })).toHaveTextContent('No session in the last 7 days.');
});
