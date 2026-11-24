// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

import { expect, test } from 'vitest';
import type { DayTokens } from '../api/types';
import { rank, segments } from './DayBars';

const days: DayTokens[] = [
  { day: '2026-11-10', tokens: { a: 10, b: 30 } },
  { day: '2026-11-11', tokens: { a: 5, c: 100, d: 30 } },
];

test('projects are ranked by total, then by name', () => {
  expect(rank(days)).toEqual(['c', 'b', 'd', 'a']);
});

test('a day is cut into the top series and one remainder', () => {
  expect(segments(days[1]!, ['c', 'b'])).toEqual([
    { key: 'c', series: 1, value: 100 },
    { key: 'others', series: 0, value: 35 },
  ]);
  expect(segments({ day: '2026-11-12', tokens: {} }, ['c', 'b'])).toEqual([]);
});
