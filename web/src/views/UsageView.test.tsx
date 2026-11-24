// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

import { fireEvent, render, screen, within } from '@testing-library/react';
import { expect, test, vi } from 'vitest';
import { stubApi } from '../test/api';
import { usage } from '../test/shapes';
import { UsageView } from './UsageView';

const month = {
  from: '2026-10-13',
  to: '2026-11-11',
  total: usage(4887),
  projects: [
    { key: 'alpha/fern', tracked: true, usage: usage(3330) },
    { key: 'other', tracked: false, usage: usage(777) },
    { key: 'alpha/gone', tracked: false, usage: usage(3) },
  ],
  models: [{ model: 'model-a', usage: usage(4332) }],
  days: [
    { day: '2026-11-04', usage: usage(3) },
    { day: '2026-11-11', usage: usage(3219) },
  ],
};

const week = { ...month, from: '2026-11-05', total: usage(4884), projects: month.projects.slice(0, 2), days: month.days.slice(1) };
const ever = { ...month, from: '2000-01-01', days: [{ day: '2025-12-30', usage: usage(9) }, ...month.days] };

const row = (key: string) => document.querySelector(`li[data-key="${key}"]`);

test('usage is shown per project and per model, for a chosen period', async () => {
  stubApi({ '/api/v1/usage': month, '/api/v1/usage?from=2026-11-05': week, '/api/v1/usage?from=2000-01-01': ever });
  const go = vi.fn();
  render(<UsageView go={go} />);

  expect(await screen.findByRole('heading', { level: 1 })).toHaveTextContent('4.9k tokens');
  expect(screen.getByText('13 October to today, 48.9k cache reads')).toBeVisible();
  // The filter stays mounted while a period loads, so it is looked up once.
  const period = within(screen.getByRole('group', { name: 'Period' }));
  expect(period.getByRole('button', { name: '30 days' })).toHaveAttribute('aria-pressed', 'true');
  // Every day of the range is drawn, the ones without usage empty.
  const chart = screen.getByRole('img', { name: 'Tokens per day' });
  expect(chart.querySelectorAll('g[data-day]')).toHaveLength(30);
  expect(chart.querySelectorAll('rect')).toHaveLength(2);
  expect(screen.getByText('at most 3.2k in a day')).toBeVisible();

  const projects = screen.getByRole('region', { name: 'Projects' });
  expect(within(projects).getAllByRole('listitem').map((li) => li.getAttribute('data-key'))).toEqual(['alpha/fern', 'other', 'alpha/gone']);
  expect(row('alpha/fern')).toHaveTextContent('3.3k usage');
  expect(row('alpha/fern')).toHaveTextContent('33.3k cache reads');
  expect(row('alpha/fern')).toHaveTextContent('1 message');
  expect(row('other')).toHaveTextContent('outside the roots');
  expect(row('alpha/gone')).toHaveTextContent('no status file');
  expect(within(projects).getAllByRole('link')).toHaveLength(1);
  fireEvent.click(within(projects).getByRole('link', { name: 'fern' }));
  expect(go).toHaveBeenCalledWith('/projects/alpha/fern');

  const models = screen.getByRole('region', { name: 'Models' });
  expect(within(models).getByRole('listitem')).toHaveTextContent('model-a');
  expect(row('model-a')).toHaveTextContent('4.3k usage');

  const seven = period.getByRole('button', { name: '7 days' });
  seven.focus();
  fireEvent.click(seven);
  expect(await screen.findByText('5 November to today, 48.8k cache reads')).toBeVisible();
  expect(seven).toBeInTheDocument();
  expect(seven).toHaveAttribute('aria-pressed', 'true');
  expect(seven).toHaveFocus();
  expect(screen.getByRole('img', { name: 'Tokens per day' }).querySelectorAll('g[data-day]')).toHaveLength(7);
  expect(row('alpha/gone')).toBeNull();

  const everything = period.getByRole('button', { name: 'everything' });
  fireEvent.click(everything);
  expect(await screen.findByText('30 December 2025 to today, 48.9k cache reads')).toBeVisible();
  expect(everything).toHaveAttribute('aria-pressed', 'true');
  // From the first day with usage to today: the range has no gap.
  expect(screen.getByRole('img', { name: 'Tokens per day' }).querySelectorAll('g[data-day]')).toHaveLength(317);
});

test('the figures stay on screen while a period loads, and the pressed period is not fetched again', async () => {
  let release!: () => void;
  const gate = new Promise<void>((resolve) => (release = resolve));
  const fetched: string[] = [];
  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: string) => {
      fetched.push(input);
      if (input !== '/api/v1/usage') await gate;
      return new Response(JSON.stringify(input === '/api/v1/usage' ? month : week), { status: 200 });
    }),
  );
  render(<UsageView go={vi.fn()} />);
  expect(await screen.findByRole('heading', { level: 1 })).toHaveTextContent('4.9k tokens');
  const period = within(screen.getByRole('group', { name: 'Period' }));

  fireEvent.click(period.getByRole('button', { name: '30 days' }));
  expect(fetched).toEqual(['/api/v1/usage']);

  fireEvent.click(period.getByRole('button', { name: '7 days' }));
  expect(fetched).toEqual(['/api/v1/usage', '/api/v1/usage?from=2026-11-05']);
  // The request is still open: the old period's figures are on screen.
  expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent('4.9k tokens');
  expect(screen.getByText('13 October to today, 48.9k cache reads')).toBeVisible();
  expect(screen.getByRole('group', { name: 'Period' })).toBeVisible();
  release();
  expect(await screen.findByText('5 November to today, 48.8k cache reads')).toBeVisible();
  vi.unstubAllGlobals();
});

test('a failed period says so above the filter and keeps the figures', async () => {
  stubApi({ '/api/v1/usage': month });
  render(<UsageView go={vi.fn()} />);
  expect(await screen.findByRole('heading', { level: 1 })).toHaveTextContent('4.9k tokens');
  const filter = screen.getByRole('group', { name: 'Period' });
  fireEvent.click(within(filter).getByRole('button', { name: '7 days' }));
  const alert = await screen.findByRole('alert');
  expect(alert).toHaveTextContent('The usage for that period could not be loaded: 404 Not Found');
  expect(alert.compareDocumentPosition(filter) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
  expect(filter).toBeInTheDocument();
  expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent('4.9k tokens');
  expect(within(filter).getByRole('button', { name: '30 days' })).toBeVisible();
});

test('the everything period without any usage prints no date range', async () => {
  const none = { ...month, from: '2000-01-01', total: usage(0), days: [], projects: [], models: [] };
  stubApi({ '/api/v1/usage': month, '/api/v1/usage?from=2000-01-01': none });
  render(<UsageView go={vi.fn()} />);
  await screen.findByRole('heading', { level: 1 });
  fireEvent.click(within(screen.getByRole('group', { name: 'Period' })).getByRole('button', { name: 'everything' }));
  expect(await screen.findByText('No usage yet.')).toBeVisible();
  expect(document.body).not.toHaveTextContent('2000');
  expect(document.body).not.toHaveTextContent('January');
});

test('single series charts and model names follow the identity', async () => {
  stubApi({ '/api/v1/usage': month });
  render(<UsageView go={vi.fn()} />);
  await screen.findByRole('heading', { level: 1 });
  for (const rect of screen.getByRole('img', { name: 'Tokens per day' }).querySelectorAll('rect')) {
    expect(rect).toHaveClass('gl-s-accent');
  }
  expect(screen.getByText('model-a')).not.toHaveClass('gl-mono');
});

test('a failed request says so', async () => {
  stubApi({});
  render(<UsageView go={vi.fn()} />);
  expect(await screen.findByRole('alert')).toHaveTextContent('The usage could not be loaded: 404 Not Found');
});

test('a period with an unpriced model is shown in tokens in every row', async () => {
  const priced = (tokens: number, cost: number) => ({ ...usage(tokens), cost });
  const mixed = {
    ...month,
    total: usage(4887),
    projects: [
      { key: 'alpha/fern', tracked: true, usage: priced(3330, 1.5) },
      { key: 'other', tracked: false, usage: usage(777) },
    ],
    models: [
      { model: 'model-a', usage: priced(4332, 2) },
      { model: 'model-b', usage: usage(555) },
    ],
  };
  stubApi({ '/api/v1/usage': mixed });
  render(<UsageView go={vi.fn()} />);
  expect(await screen.findByRole('heading', { level: 1 })).toHaveTextContent('4.9k tokens');
  expect(row('alpha/fern')).toHaveTextContent('3.3k usage');
  expect(row('other')).toHaveTextContent('777 usage');
  expect(row('model-a')).toHaveTextContent('4.3k usage');
  expect(row('model-b')).toHaveTextContent('555 usage');
  expect(document.body).not.toHaveTextContent('$');
});

test('a period where every model is priced is shown as a cost in every row', async () => {
  const priced = (tokens: number, cost: number) => ({ ...usage(tokens), cost });
  const all = {
    ...month,
    total: priced(4887, 3.5),
    projects: [{ key: 'alpha/fern', tracked: true, usage: priced(3330, 1.5) }],
    models: [{ model: 'model-a', usage: priced(4332, 2) }],
  };
  stubApi({ '/api/v1/usage': all });
  render(<UsageView go={vi.fn()} />);
  expect(await screen.findByRole('heading', { level: 1 })).toHaveTextContent('$3.50');
  expect(row('alpha/fern')).toHaveTextContent('$1.50 usage');
  expect(row('model-a')).toHaveTextContent('$2.00 usage');
});
