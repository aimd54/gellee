// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

import { fireEvent, render, screen, within } from '@testing-library/react';
import { expect, test, vi } from 'vitest';
import { Inline } from './md';
import { Link, PageHead, Pill, Segmented, StatePill, StateRow, WarningPills } from './parts';
import { project } from './test/shapes';

test('a link routes a plain click and leaves a modified one to the browser', () => {
  const go = vi.fn();
  render(
    <Link to="/ideas" go={go}>
      Ideas
    </Link>,
  );
  const a = screen.getByRole('link', { name: 'Ideas' });
  expect(a).toHaveAttribute('href', '/ideas');
  fireEvent.click(a, { ctrlKey: true });
  expect(go).not.toHaveBeenCalled();
  fireEvent.click(a);
  expect(go).toHaveBeenCalledWith('/ideas');
});

test('the page head holds the small line, the heading and its sentence', () => {
  render(
    <PageHead eyebrow="Wednesday, 11 November" title="Pick up the thread." after={<Pill tone="late">1 over the limit</Pill>}>
      3 projects on the bench.
    </PageHead>,
  );
  expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent('Pick up the thread.');
  expect(screen.getByText('Wednesday, 11 November')).toHaveClass('gl-eyebrow');
  expect(screen.getByText('3 projects on the bench.')).toHaveClass('gl-lede');
  expect(screen.getByText('1 over the limit')).toHaveClass('gl-pill', 'gl-tone-late');
});

test('state and warnings are pills with a word and a tone', () => {
  render(
    <>
      <StatePill state="active" />
      <StatePill state="parked" />
      <StatePill state="done" />
      <WarningPills names={['stale', 'unpushed', 'uncommitted']} />
    </>,
  );
  expect(screen.getByText('active')).toHaveClass('gl-tone-accent');
  expect(screen.getByText('parked')).toHaveClass('gl-tone-unknown');
  expect(screen.getByText('done')).toHaveClass('gl-tone-done');
  for (const word of ['stale status', 'unpushed work', 'uncommitted work']) {
    expect(screen.getByText(word)).toHaveClass('gl-tone-late');
  }
});

test('a segmented filter shows counts, one choice pressed, and reports a click', () => {
  const onChange = vi.fn();
  render(
    <Segmented
      label="State"
      value="all"
      onChange={onChange}
      choices={[
        { value: 'all', label: 'All', count: 5 },
        { value: 'parked', label: 'Parked', count: 1 },
        { value: 'plain', label: 'Plain' },
      ]}
    />,
  );
  const group = screen.getByRole('group', { name: 'State' });
  expect(within(group).getByRole('button', { name: 'All 5' })).toHaveAttribute('aria-pressed', 'true');
  expect(within(group).getByRole('button', { name: 'Parked 1' })).toHaveAttribute('aria-pressed', 'false');
  expect(within(group).getByRole('button', { name: 'Plain' })).toBeVisible();
  fireEvent.click(within(group).getByRole('button', { name: 'Parked 1' }));
  expect(onChange).toHaveBeenCalledWith('parked');
});

test('a project row is one link with the name, one line, a figure, the state and the date', () => {
  const go = vi.fn();
  render(
    <ul>
      <StateRow
        project={project({ key: 'alpha/fern', updated: '2026-11-09' })}
        today="2026-11-11"
        shared={new Set()}
        go={go}
        line="Merge `m2-task-17`."
        figure={
          <>
            <b>3.3k</b> 30 days
          </>
        }
      />
    </ul>,
  );
  const row = within(screen.getByRole('listitem')).getByRole('link');
  expect(row).toHaveAttribute('href', '/projects/alpha/fern');
  expect(row).toHaveAttribute('data-project', 'alpha/fern');
  expect(within(row).getByText('fern')).toBeVisible();
  expect(row).toHaveTextContent('Merge m2-task-17.');
  expect(within(row).getByText('m2-task-17').tagName).toBe('CODE');
  expect(row).toHaveTextContent('3.3k 30 days');
  expect(within(row).getByText('active')).toBeVisible();
  expect(within(row).getByText('9 November')).toBeVisible();
  fireEvent.click(row);
  expect(go).toHaveBeenCalledWith('/projects/alpha/fern');
});

test('in a project row, warnings take the place of the date, and a shared name carries its root', () => {
  render(
    <ul>
      <StateRow
        project={project({ key: 'alpha/tool', updated: '2026-11-09', warnings: ['stale'] })}
        today="2026-11-11"
        shared={new Set(['tool'])}
        go={vi.fn()}
        line="Run the review."
      />
    </ul>,
  );
  const row = screen.getByRole('link');
  expect(within(row).getByText('tool (alpha)')).toBeVisible();
  expect(within(row).getByText('stale status')).toBeVisible();
  expect(within(row).queryByText('9 November')).toBeNull();
});

test('inline text renders backticks as code', () => {
  render(
    <p data-testid="p">
      <Inline text="On `main`, clean." />
    </p>,
  );
  expect(screen.getByTestId('p')).toHaveTextContent('On main, clean.');
  expect(screen.getByText('main').tagName).toBe('CODE');
});

test('a project row tiles active in the accent, parked and done in grey, and groups its warnings', () => {
  const go = vi.fn();
  const rows = [
    project({ key: 'alpha/a', state: 'active', warnings: ['stale', 'unpushed'] }),
    project({ key: 'alpha/b', state: 'parked' }),
    project({ key: 'alpha/c', state: 'done' }),
  ];
  render(
    <ul>
      {rows.map((p) => (
        <StateRow key={p.key} project={p} today="2026-11-11" shared={new Set()} go={go} line="x" />
      ))}
    </ul>,
  );
  const tile = (key: string) => document.querySelector(`a[data-project="${key}"] .gl-tile`)!;
  expect(tile('alpha/a')).toHaveClass('gl-tone-accent');
  expect(tile('alpha/b')).toHaveClass('gl-tone-unknown');
  expect(tile('alpha/c')).toHaveClass('gl-tone-unknown');
  // The done state keeps its own colour on its pill.
  expect(screen.getByText('done')).toHaveClass('gl-tone-done');
  // Warnings share one line under the state, so a row with two stays as
  // tall as one with a single pill and a date.
  const warns = document.querySelector('a[data-project="alpha/a"] .gl-row-warns')!;
  expect(warns).toHaveTextContent('stale statusunpushed work');
  expect(warns.querySelectorAll('.gl-pill')).toHaveLength(2);
});
