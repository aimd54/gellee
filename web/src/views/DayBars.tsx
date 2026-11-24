// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

import type { DayTokens } from '../api/types';
import { readerDate } from '../dates';
import { tokens as fmt } from '../format';
import { label, sharedNames } from '../project';

export const SERIES = 6;
export const REST = 'others';

export type Segment = { key: string; series: number; value: number };

// rank orders the projects of a period by their total, largest first.
export function rank(days: DayTokens[]): string[] {
  const totals = new Map<string, number>();
  for (const d of days) for (const [k, v] of Object.entries(d.tokens)) totals.set(k, (totals.get(k) ?? 0) + v);
  return [...totals.entries()].sort((a, b) => b[1] - a[1] || a[0].localeCompare(b[0])).map(([k]) => k);
}

// segments cuts one day into the top projects, numbered from 1, and a
// remainder numbered 0.
export function segments(day: DayTokens, top: string[]): Segment[] {
  const out: Segment[] = [];
  top.forEach((key, i) => {
    const value = day.tokens[key] ?? 0;
    if (value > 0) out.push({ key, series: i + 1, value });
  });
  const rest = Object.entries(day.tokens)
    .filter(([k]) => !top.includes(k))
    .reduce((sum, [, v]) => sum + v, 0);
  if (rest > 0) out.push({ key: REST, series: 0, value: rest });
  return out;
}

const W = 720;
const H = 160;
const GAP = 2;

// gapFor shrinks the gap between bars as the days grow, so that a bar
// always keeps a positive width.
function gapFor(n: number): number {
  return Math.min(GAP, W / n / 4);
}

export function DayBars({
  days,
  label: title,
  today,
  legend = true,
  accent = false,
  range = true,
}: {
  days: DayTokens[];
  label: string;
  today: string;
  legend?: boolean;
  // accent draws one series in the accent colour, for a chart of a
  // single total. The series colours are for the stacked chart only.
  accent?: boolean;
  // range prints the first and last day under the chart; a view whose
  // heading states the range leaves it out.
  range?: boolean;
}) {
  const ranked = rank(days);
  if (ranked.length === 0) return <p className="gl-muted">No usage in this period.</p>;
  const top = ranked.slice(0, SERIES);
  const shared = sharedNames(ranked);
  const peak = Math.max(1, ...days.map((d) => Object.values(d.tokens).reduce((a, b) => a + b, 0)));
  const gap = gapFor(days.length);
  const width = (W - gap * (days.length - 1)) / days.length;
  return (
    <figure className="gl-chart">
      <svg viewBox={`0 0 ${W} ${H}`} preserveAspectRatio="none" role="img" aria-label={title}>
        {days.map((d, i) => {
          const segs = segments(d, top);
          const heights = segs.map((s) => (s.value / peak) * H);
          return (
            <g key={d.day} data-day={d.day}>
              {segs.map((s, j) => {
                const below = heights.slice(0, j).reduce((a, b) => a + b, 0);
                return (
                  <rect
                    key={s.key}
                    className={accent ? 'gl-s-accent' : s.series ? `gl-s${s.series}` : 'gl-s-rest'}
                    x={i * (width + gap)}
                    y={H - below - heights[j]!}
                    width={width}
                    height={heights[j]!}
                  >
                    <title>{`${readerDate(d.day, today)}, ${s.key === REST ? REST : label(s.key, shared)}: ${fmt(s.value)}`}</title>
                  </rect>
                );
              })}
            </g>
          );
        })}
      </svg>
      <figcaption className="gl-legend">
        {legend &&
          top.map((k, i) => (
            <span key={k}>
              <i className={`gl-s${i + 1}`} />
              {label(k, shared)}
            </span>
          ))}
        {legend && ranked.length > SERIES && (
          <span>
            <i className="gl-s-rest" />
            {REST}
          </span>
        )}
        <span className="gl-muted">
          {`${range ? `${readerDate(days[0]!.day, today)} to ${readerDate(days[days.length - 1]!.day, today)}, ` : ''}at most ${fmt(peak)} in a day`}
        </span>
      </figcaption>
    </figure>
  );
}
