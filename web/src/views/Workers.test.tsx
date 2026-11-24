// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

import { fireEvent, render, screen, within } from '@testing-library/react';
import { expect, test, vi } from 'vitest';
import { stubApi } from '../test/api';
import { session, worker } from '../test/shapes';
import { seenLine, Workers } from './Workers';

const today = '2026-11-11';

const tracked = (over: Parameters<typeof session>[0], tracked = true) => ({ ...session(over), tracked });

const data = {
  built: '2026-11-11T12:00:00Z',
  today,
  problems: [],
  workers: [
    worker({
      name: 'm1', collected: '2026-11-11T10:00:00+01:00', up: true, claude_code: '2.1.323',
      memories: { pushed: '2026-11-11T09:50:00+01:00', set_aside: 1, stale: false },
      sessions: [
        tracked({ id: 'a', project: 'beta/tool', title: 'Parse nested lists', remote: 'session_01Tool', last: '2026-11-11T09:55:00+01:00' }),
        tracked({ id: 'b', project: 'other', title: '', branch: '', last: '2026-11-10T12:30:00+01:00' }, false),
        tracked({ id: 'c', project: 'beta/scratch', title: 'Try a parser', last: '2026-11-10T11:30:00+01:00' }, false),
        tracked({
          id: 'd', project: 'beta/tool', title: 'Archived run', branch: 'worktree-bridge-cse_01AVeryLongBridgeIdentifier',
          remote: 'session_01Done', last: '2026-11-10T10:00:00+01:00', ended: true,
        }),
      ],
      checkouts: [
        { project: 'alpha/fern', branch: 'm2-task-17', warnings: ['stale', 'unpushed'] },
        { project: 'beta/tool', branch: 'main', warnings: [] },
      ],
    }),
    worker({ name: 'm3', collected: '2026-11-11T07:00:00+01:00', up: false, claude_code: '', memories: null }),
    worker({
      name: 'm2', collected: '2026-11-09T18:00:00+01:00', up: false, claude_code: '2.1.300', behind: true,
      memories: { pushed: '2026-11-08T09:00:00+01:00', set_aside: 0, stale: true },
    }),
  ],
};

const card = (name: string) => screen.getByRole('region', { name });

test('the head says every figure is from the last collection, and cards come in the order served, most recently seen first', async () => {
  stubApi({ '/api/v1/workers': data });
  render(<Workers go={vi.fn()} />);
  expect(await screen.findByRole('heading', { level: 1, name: 'Workers' })).toBeVisible();
  expect(screen.getByText("Every figure is from each worker's last collection.")).toBeVisible();
  expect(screen.getAllByRole('region').map((r) => r.getAttribute('aria-label'))).toEqual(['m1', 'm3', 'm2']);
});

test('a card says when the worker was seen, its version and its memory sync', async () => {
  stubApi({ '/api/v1/workers': data });
  render(<Workers go={vi.fn()} />);
  await screen.findByRole('region', { name: 'm1' });
  expect(card('m1')).toHaveTextContent('seen today, 10:00');
  expect(card('m1')).toHaveTextContent('Claude Code 2.1.323');
  expect(card('m1')).toHaveTextContent('memories pushed today, 09:50');
  expect(card('m1')).toHaveTextContent('1 version set aside');
  expect(within(card('m1')).queryByText('behind')).toBeNull();
  expect(within(card('m1')).queryByText('sync stale')).toBeNull();

  expect(card('m2')).toHaveTextContent('not seen since 9 November');
  expect(within(card('m2')).getByText('behind')).toHaveClass('gl-tone-late');
  expect(within(card('m2')).getByText('sync stale')).toHaveClass('gl-tone-late');
  expect(card('m2')).not.toHaveTextContent('set aside');

  expect(card('m3')).toHaveTextContent('not seen since today, 07:00');
  expect(card('m3')).toHaveTextContent('version unknown');
  expect(card('m3')).not.toHaveTextContent('memories');
});

test('sessions link their project and open in Claude; other is not linked', async () => {
  stubApi({ '/api/v1/workers': data });
  const go = vi.fn();
  render(<Workers go={go} />);
  const rows = within(await screen.findByRole('region', { name: 'm1' })).getAllByRole('listitem');
  expect(rows[0]).toHaveTextContent('Parse nested lists');
  expect(within(rows[0]!).getByRole('link', { name: /^Open\b/ })).toHaveAttribute('href', 'https://claude.ai/code/session_01Tool');
  fireEvent.click(within(rows[0]!).getByRole('link', { name: 'tool' }));
  expect(go).toHaveBeenCalledWith('/projects/beta/tool');
  expect(rows[1]).toHaveTextContent('other');
  expect(within(rows[1]!).queryByRole('link')).toBeNull();
  expect(card('m2')).toHaveTextContent('No session in the last 7 days.');
});

test('a session of a project with no status file shows its name and a note, not a link', async () => {
  stubApi({ '/api/v1/workers': data });
  render(<Workers go={vi.fn()} />);
  const rows = within(await screen.findByRole('region', { name: 'm1' })).getAllByRole('listitem');
  expect(rows[2]).toHaveTextContent('scratch');
  expect(rows[2]).toHaveTextContent('no status file');
  expect(within(rows[2]!).queryByRole('link', { name: 'scratch' })).toBeNull();
  expect(rows[0]).not.toHaveTextContent('no status file');
});

test('ended sessions are folded apart from the others until asked for', async () => {
  stubApi({ '/api/v1/workers': data });
  render(<Workers go={vi.fn()} />);
  const m1 = await screen.findByRole('region', { name: 'm1' });
  expect(within(m1).getAllByRole('listitem')).toHaveLength(3);
  const fold = within(m1).getByRole('button', { name: 'Ended (1)' });
  expect(fold).toHaveAttribute('aria-expanded', 'false');
  expect(within(m1).queryByText('Archived run')).toBeNull();
  fireEvent.click(fold);
  expect(fold).toHaveAttribute('aria-expanded', 'true');
  const row = within(m1).getByText('Archived run').closest('li')!;
  expect(within(row).getByRole('link', { name: /^Open\b/ })).toHaveAttribute('href', 'https://claude.ai/code/session_01Done');
  expect(within(card('m2')).queryByRole('button', { name: /^Ended/ })).toBeNull();
});

test('a worker whose sessions all ended shows only the fold', async () => {
  stubApi({
    '/api/v1/workers': {
      ...data,
      workers: [worker({ name: 'm4', sessions: [tracked({ id: 'e', ended: true })] })],
    },
  });
  render(<Workers go={vi.fn()} />);
  const m4 = await screen.findByRole('region', { name: 'm4' });
  expect(within(m4).getByRole('button', { name: 'Ended (1)' })).toBeVisible();
  expect(m4).not.toHaveTextContent('No session in the last 7 days.');
  expect(within(m4).queryAllByRole('listitem')).toHaveLength(0);
});

test('a long branch is cut short with its full name on hover', async () => {
  stubApi({ '/api/v1/workers': data });
  render(<Workers go={vi.fn()} />);
  const m1 = await screen.findByRole('region', { name: 'm1' });
  fireEvent.click(within(m1).getByRole('button', { name: 'Ended (1)' }));
  const branch = within(m1).getByText('worktree-bridge-cse_01AVeryLongBridgeIdentifier');
  expect(branch).toHaveClass('gl-session-branch');
  expect(branch).toHaveAttribute('title', 'worktree-bridge-cse_01AVeryLongBridgeIdentifier');
});

test('checkouts are folded until asked for', async () => {
  stubApi({ '/api/v1/workers': data });
  render(<Workers go={vi.fn()} />);
  const m1 = await screen.findByRole('region', { name: 'm1' });
  const fold = within(m1).getByRole('button', { name: 'Checkouts (2)' });
  expect(fold).toHaveAttribute('aria-expanded', 'false');
  expect(within(m1).queryByText('unpushed work')).toBeNull();
  fireEvent.click(fold);
  expect(fold).toHaveAttribute('aria-expanded', 'true');
  expect(within(m1).getByText('m2-task-17')).toBeVisible();
  expect(within(m1).getByText('unpushed work')).toBeVisible();
  expect(within(card('m2')).queryByRole('button')).toBeNull();
});

test('seenLine', () => {
  expect(seenLine(worker({ collected: '', up: false }), today)).toBe('no collection recorded');
  expect(seenLine(worker({ collected: '2026-11-10T22:00:00+01:00', up: false }), today)).toBe('not seen since yesterday');
});
