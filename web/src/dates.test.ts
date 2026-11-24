// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

import { expect, test } from 'vitest';
import { collectedLine, daysBetween, longDay, readerDate, readerMoment } from './dates';

const today = '2026-11-11';

test('a date reads as today, yesterday, then a day and a month', () => {
  expect(readerDate('2026-11-11', today)).toBe('today');
  expect(readerDate('2026-11-10', today)).toBe('yesterday');
  expect(readerDate('2026-11-09', today)).toBe('9 November');
  expect(readerDate('2026-10-20', today)).toBe('20 October');
});

test('the year is added when it differs from the current one', () => {
  expect(readerDate('2025-11-09', today)).toBe('9 November 2025');
  expect(readerDate('2025-12-31', '2026-01-01')).toBe('yesterday');
  expect(readerDate('2025-12-30', '2026-01-01')).toBe('30 December 2025');
});

test('text that is not a date never reaches the page', () => {
  expect(readerDate('soon', today)).toBe('an unknown date');
  expect(readerDate('11/09/2026', today)).toBe('an unknown date');
});

test('days are counted across months', () => {
  expect(daysBetween('2026-10-29', '2026-11-10')).toBe(12);
  expect(daysBetween('2026-11-09', '2026-11-10')).toBe(1);
});

test('a moment keeps the clock time its machine wrote', () => {
  expect(readerMoment('2026-11-10T09:00:00+01:00', today)).toBe('yesterday, 09:00');
  expect(readerMoment('2026-11-09T21:43:10Z', today)).toBe('9 November, 21:43');
});

test('the long day names the weekday', () => {
  expect(longDay('2026-11-11')).toBe('Wednesday, 11 November');
  expect(longDay('2026-03-01')).toBe('Sunday, 1 March');
});

test('the collection line gives the time today and the date before', () => {
  expect(collectedLine('2026-11-11T10:00:00+01:00', today)).toBe('collected 10:00');
  expect(collectedLine('2026-11-09T23:10:00+01:00', today)).toBe('collected 9 November');
  expect(collectedLine('', today)).toBe('not collected yet');
});
