// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

import { expect, test } from 'vitest';
import { addDays, amount, cost, count, figure, figures, tokens } from './format';
import { usage } from './test/shapes';

test('tokens are shortened with one decimal below a hundred units', () => {
  expect(tokens(0)).toBe('0');
  expect(tokens(999)).toBe('999');
  expect(tokens(1000)).toBe('1k');
  expect(tokens(3330)).toBe('3.3k');
  expect(tokens(14430)).toBe('14.4k');
  expect(tokens(2_500_000)).toBe('2.5M');
  expect(tokens(123_456_789)).toBe('123M');
  expect(tokens(4_200_000_000)).toBe('4.2B');
});

test('cost has two decimals and never rounds a real amount to zero', () => {
  expect(cost(12.345)).toBe('$12.35');
  expect(cost(0.004)).toBe('<$0.01');
  expect(cost(0)).toBe('$0.00');
});

test('the figure is the cost when there is one, tokens otherwise', () => {
  expect(figure(usage(3330))).toBe('3.3k');
  expect(figure({ ...usage(3330), cost: 1.5 })).toBe('$1.50');
});

test('addDays crosses months', () => {
  expect(addDays('2026-11-11', -29)).toBe('2026-10-13');
  expect(addDays('2026-02-28', 1)).toBe('2026-03-01');
});

test('a count carries its noun, singular for one', () => {
  expect(count(1, 'file', 'files')).toBe('1 file');
  expect(count(3, 'file', 'files')).toBe('3 files');
  expect(count(0, 'file', 'files')).toBe('0 files');
});

test('an amount names its unit unless it is a cost', () => {
  expect(amount(usage(4887))).toBe('4.9k tokens');
  expect(amount({ ...usage(4887), cost: 1.5 })).toBe('$1.50');
});

test('figures shown together share one unit', () => {
  const priced = { ...usage(1200), cost: 0.12 };
  const unpriced = usage(3330);
  const nothing = { ...usage(0), messages: 0 };

  expect([priced, { ...priced, cost: 2 }].map(figures([priced, { ...priced, cost: 2 }]))).toEqual(['$0.12', '$2.00']);
  // A sum that includes a model without a price puts the whole set in tokens.
  expect([priced, unpriced].map(figures([priced, unpriced]))).toEqual(['1.2k', '3.3k']);
  // A sum of nothing has no cost of its own and does not decide the unit.
  expect([priced, nothing].map(figures([priced, nothing]))).toEqual(['$0.12', '$0.00']);
  expect([unpriced, nothing].map(figures([unpriced, nothing]))).toEqual(['3.3k', '0']);
  expect([nothing].map(figures([nothing]))).toEqual(['0']);
  expect(figures([])(unpriced)).toBe('3.3k');
});
