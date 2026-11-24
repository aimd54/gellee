// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

import { render, screen } from '@testing-library/react';
import { expect, test } from 'vitest';
import { addDays } from '../format';
import { DayBars } from './DayBars';

const run = (n: number) =>
  Array.from({ length: n }, (_, i) => ({ day: addDays('2025-01-01', i), tokens: { a: i % 7 === 0 ? 5 : 1 } }));

test('every bar has a width, however many days are drawn', () => {
  for (const n of [1, 30, 90, 360, 400, 1200]) {
    const { unmount } = render(<DayBars days={run(n)} label="Tokens" today="2027-01-01" />);
    const rects = screen.getByRole('img', { name: 'Tokens' }).querySelectorAll('rect');
    expect(rects.length, `${n} days`).toBe(n);
    for (const r of rects) expect(Number(r.getAttribute('width')), `${n} days`).toBeGreaterThan(0);
    const last = rects[n - 1]!;
    expect(Number(last.getAttribute('x')) + Number(last.getAttribute('width')), `${n} days`).toBeLessThanOrEqual(720.0001);
    unmount();
  }
});

test('a single series is drawn in the accent', () => {
  render(<DayBars days={run(3)} label="Tokens" today="2026-01-01" accent legend={false} />);
  for (const r of screen.getByRole('img', { name: 'Tokens' }).querySelectorAll('rect')) {
    expect(r).toHaveClass('gl-s-accent');
    expect(r).not.toHaveClass('gl-s1');
  }
});

test('the range can be left to the heading', () => {
  render(<DayBars days={run(3)} label="Tokens" today="2026-01-01" range={false} legend={false} />);
  expect(screen.getByText('at most 5 in a day')).toBeVisible();
});
