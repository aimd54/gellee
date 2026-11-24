// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

import { useState, type ReactNode } from 'react';
import type { Usage, UsageData } from '../api/types';
import { useApi } from '../api/useApi';
import { daysBetween, readerDate } from '../dates';
import { addDays, amount, count, figures, tokens } from '../format';
import { Link, PageHead, Pending, Segmented, type Go } from '../parts';
import { label, projectPath, sharedNames } from '../project';
import { DayBars } from './DayBars';

// EVERYTHING_FROM is the date the "everything" period asks from.
const EVERYTHING_FROM = '2000-01-01';

const PERIODS = [
  { value: '7 days', days: 7 },
  { value: '30 days', days: 30 },
  { value: '90 days', days: 90 },
  { value: 'everything', days: null },
] as const;
type Period = (typeof PERIODS)[number]['value'];

function Row({ dataKey, name, note, usage, figure }: { dataKey: string; name: ReactNode; note?: string; usage: Usage; figure: string }) {
  return (
    <li className="gl-row" data-key={dataKey}>
      <span className="gl-row-text">
        <b>{name}</b>
        {note && <span className="gl-row-line">{note}</span>}
      </span>
      <span className="gl-row-figure">
        <b>{figure}</b> usage
      </span>
      <span className="gl-row-figure">
        <b>{tokens(usage.cache_read)}</b> cache reads
      </span>
      <span className="gl-row-figure">
        <b>{usage.messages}</b> {usage.messages === 1 ? 'message' : 'messages'}
      </span>
    </li>
  );
}

export function UsageView({ go }: { go: Go }) {
  // The server knows today's date, so the first request names no period
  // and later ones count back from the "to" it answered with. Every
  // period ends today, which makes "to" the server's date.
  const [chosen, setChosen] = useState<{ period: Period; from: string | null }>({ period: '30 days', from: null });
  // The previous period stays on screen while the next one loads, so the
  // filter keeps its place and its focus.
  const { data, error, pending } = useApi<UsageData>(chosen.from ? `/usage?from=${chosen.from}` : '/usage', true);
  if (!data) return <Pending error={error} what="usage" />;
  const today = data.to;
  // "everything" asks from a date long past: its range starts at the
  // first day that has usage, and is not stated when there is none.
  const everything = data.from === EVERYTHING_FROM;
  const first = everything ? data.days[0]?.day : data.from;
  const perDay = new Map(data.days.map((d) => [d.day, d.usage.tokens]));
  // The server answers only the days that have usage: the range is
  // filled in here so every day takes its own place on the axis.
  const span = first === undefined ? 0 : Math.max(0, daysBetween(first, today) + 1);
  const drawn = Array.from({ length: span }, (_, i) => {
    const day = addDays(first!, i);
    return { day, tokens: { 'all projects': perDay.get(day) ?? 0 } };
  });
  const shared = sharedNames(data.projects.map((p) => p.key));
  // The total includes every model of the period, so it settles the unit
  // of both lists.
  const figure = figures([data.total]);
  const choose = (period: Period) => {
    // The pressed period is already on screen; a failed one may be tried again.
    if (period === chosen.period && error === null) return;
    const days = PERIODS.find((p) => p.value === period)!.days;
    setChosen({ period, from: days === null ? EVERYTHING_FROM : addDays(today, -(days - 1)) });
  };
  return (
    <>
      <PageHead eyebrow="Usage" title={amount(data.total)}>
        {first === undefined
          ? 'No usage yet.'
          : `${readerDate(first, today)} to ${readerDate(data.to, today)}, ${tokens(data.total.cache_read)} cache reads`}
      </PageHead>
      {error !== null && (
        <p role="alert" className="gl-problems">
          The usage for that period could not be loaded: {error}
        </p>
      )}
      <div className="gl-filters">
        <Segmented label="Period" value={chosen.period} onChange={choose} choices={PERIODS.map((p) => ({ value: p.value, label: p.value }))} />
      </div>
      <div aria-busy={pending}>
        <section className="gl-card gl-pad" aria-label="Per day">
          <DayBars days={drawn} label="Tokens per day" today={today} legend={false} accent range={false} />
        </section>
        <section aria-labelledby="gl-usage-projects">
          <div className="gl-grp">
            <h2 id="gl-usage-projects">Projects</h2>
            <span className="gl-meta">{count(data.projects.length, 'project', 'projects')}</span>
          </div>
          <ul className="gl-card gl-rows">
            {data.projects.map((p) => (
              <Row
                key={p.key}
                dataKey={p.key}
                name={
                  p.tracked ? (
                    <Link to={projectPath(p.key)} go={go} className="gl-link">
                      {label(p.key, shared)}
                    </Link>
                  ) : (
                    label(p.key, shared)
                  )
                }
                note={p.tracked ? undefined : p.key === 'other' ? 'outside the roots' : 'no status file'}
                usage={p.usage}
                figure={figure(p.usage)}
              />
            ))}
          </ul>
        </section>
        <section aria-labelledby="gl-usage-models">
          <div className="gl-grp">
            <h2 id="gl-usage-models">Models</h2>
            <span className="gl-meta">{count(data.models.length, 'model', 'models')}</span>
          </div>
          <ul className="gl-card gl-rows">
            {data.models.map((m) => (
              <Row key={m.model} dataKey={m.model} name={m.model} usage={m.usage} figure={figure(m.usage)} />
            ))}
          </ul>
        </section>
      </div>
    </>
  );
}
