// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

import type { Usage } from './api/types';

function short(v: number): string {
  return v >= 100 ? v.toFixed(0) : v.toFixed(1).replace(/\.0$/, '');
}

export function tokens(n: number): string {
  if (n < 1_000) return String(n);
  if (n < 1_000_000) return `${short(n / 1_000)}k`;
  if (n < 1_000_000_000) return `${short(n / 1_000_000)}M`;
  return `${short(n / 1_000_000_000)}B`;
}

export function cost(c: number): string {
  return c > 0 && c < 0.005 ? '<$0.01' : `$${c.toFixed(2)}`;
}

// The headline figure: an estimated cost when prices are configured for
// every model involved, tokens without cache reads otherwise.
export function figure(u: Usage): string {
  return u.cost !== null ? cost(u.cost) : tokens(u.tokens);
}

export function addDays(day: string, n: number): string {
  const d = new Date(`${day}T00:00:00Z`);
  d.setUTCDate(d.getUTCDate() + n);
  return d.toISOString().slice(0, 10);
}

// count writes a number with its noun: "1 file", "3 files".
export function count(n: number, one: string, many: string): string {
  return `${n} ${n === 1 ? one : many}`;
}

// amount is the headline figure with its unit: "4.9k tokens", or the
// estimated cost as it is.
export function amount(u: Usage): string {
  return u.cost !== null ? cost(u.cost) : `${tokens(u.tokens)} tokens`;
}

// figures gives the formatter for sums that are shown side by side, so
// that they share one unit: the estimated cost when every sum is priced,
// tokens as soon as one of them includes a model without a price. A sum
// of nothing has no cost of its own and takes the unit of the others.
export function figures(all: Usage[]): (u: Usage) => string {
  const empty = (u: Usage) => u.input + u.output + u.cache_write + u.cache_read === 0;
  const priced = all.some((u) => u.cost !== null) && all.every((u) => u.cost !== null || empty(u));
  return (u) => (priced ? cost(u.cost ?? 0) : tokens(u.tokens));
}
