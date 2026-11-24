// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

import { ChevronDown, ChevronRight } from 'lucide-react';
import { Fragment, useId, useState } from 'react';
import type { DetailData, GitProject, HealthData } from '../api/types';
import { useApi } from '../api/useApi';
import { readerDate, readerMoment } from '../dates';
import { addDays, count, figures, tokens } from '../format';
import { Markdown } from '../md';
import { Fold, Link, PageHead, Pending, StatePill, WarningPills, type Go } from '../parts';
import { groupId, label, nameOf, projectPath, rootOf, sharedNames } from '../project';
import { SessionRows } from '../sessions';
import { DayBars } from './DayBars';

const FOLD = 'Day by day, per model';

// dominant is the model that accounts for the most tokens.
export function dominant(days: DetailData['days']): string | null {
  const totals = new Map<string, number>();
  for (const d of days) totals.set(d.model, (totals.get(d.model) ?? 0) + d.usage.tokens);
  const ranked = [...totals.entries()].sort((a, b) => b[1] - a[1] || a[0].localeCompare(b[0]));
  return ranked[0]?.[0] ?? null;
}

function Block({ title, text, empty, onRail }: { title: string; text: string; empty: string; onRail?: boolean }) {
  return (
    <section className={onRail ? 'gl-card gl-block gl-on-rail' : 'gl-card gl-block'} aria-label={title}>
      <h2 className="gl-eyebrow">{title}</h2>
      {text ? (
        <div className="gl-prose">
          <Markdown text={text} />
        </div>
      ) : (
        <p className="gl-muted">{empty}</p>
      )}
    </section>
  );
}

// GitCard shows what one machine saw. A figure that raises a warning
// takes the warning colour.
function GitCard({ machine, git: g, updated, today }: { machine: string; git: GitProject; updated: string; today: string }) {
  const stale = g.last_commit.slice(0, 10) > updated;
  const commits = count(g.ahead, 'commit', 'commits');
  return (
    <section className="gl-card gl-side-card" aria-label={`Git, ${machine}`}>
      <h2 className="gl-eyebrow">{`Git, ${machine}`}</h2>
      <dl className="gl-kv">
        <dt>Branch</dt>
        <dd className={g.branch === '' ? undefined : 'gl-mono'}>{g.branch === '' ? 'detached' : g.branch}</dd>
        <dt>Last commit</dt>
        <dd className={stale ? 'gl-late' : undefined}>
          {g.last_commit === '' ? (
            'no commits'
          ) : (
            <>
              {readerMoment(g.last_commit, today)} <span className="gl-mono">{g.head}</span>
            </>
          )}
        </dd>
        <dt>Unpushed</dt>
        <dd className={g.ahead > 0 || !g.remote ? 'gl-late' : undefined}>
          {!g.remote ? (g.ahead > 0 ? `${commits}, no remote` : 'no remote') : g.ahead > 0 ? commits : 'nothing'}
        </dd>
        <dt>Uncommitted</dt>
        <dd className={g.dirty > 0 ? 'gl-late' : undefined}>{g.dirty > 0 ? count(g.dirty, 'file', 'files') : 'nothing'}</dd>
      </dl>
    </section>
  );
}

export function ProjectPage({ projectKey, go }: { projectKey: string; go: Go }) {
  const detail = useApi<DetailData>(projectPath(projectKey));
  const health = useApi<HealthData>('/health');
  const [open, setOpen] = useState(false);
  const tableId = useId();
  if (detail.error?.startsWith('404')) {
    return (
      <>
        <PageHead title="No project at this address" />
        <Link to="/projects" go={go} className="gl-link">
          Back to Projects
        </Link>
      </>
    );
  }
  if (!detail.data || !health.data) return <Pending error={detail.error ?? health.error} what="project" />;

  const { project: p, group_members: members, days, sessions } = detail.data;
  const today = health.data.today;
  const live = sessions.filter((x) => !x.ended);
  const ended = sessions.filter((x) => x.ended);
  const s = p.sections ?? { stopped: '', next: '', remaining: '', ideas: '' };
  const shared = sharedNames([p.key, ...members, p.superseded_by].filter(Boolean));
  const machines = Object.keys(p.git).sort();
  const model = dominant(days);
  // A cost is shown only when every figure beside it has one: a priced
  // model next to an unpriced one would put dollars next to tokens.
  const figure = figures([p.usage_7, p.usage_30, p.usage_all]);
  const dayFigure = figures(days.map((d) => d.usage));

  const perDay = new Map<string, number>();
  for (const d of days) perDay.set(d.day, (perDay.get(d.day) ?? 0) + d.usage.tokens);
  const month = Array.from({ length: 30 }, (_, i) => {
    const day = addDays(today, i - 29);
    const n = perDay.get(day) ?? 0;
    const tokensOfDay: Record<string, number> = n > 0 ? { [p.key]: n } : {};
    return { day, tokens: tokensOfDay };
  });

  return (
    <>
      <nav className="gl-crumb" aria-label="Breadcrumb">
        <Link to="/projects" go={go} className="gl-quiet">
          Projects
        </Link>
        <ChevronRight size={14} aria-hidden="true" />
        {p.group && (
          <>
            <Link to={`/projects#${groupId(p.group)}`} go={go} className="gl-quiet">
              {p.group}
            </Link>
            <ChevronRight size={14} aria-hidden="true" />
          </>
        )}
        <span aria-current="page">{nameOf(p.key)}</span>
      </nav>
      <PageHead
        title={nameOf(p.key)}
        after={
          <>
            <StatePill state={p.state} />
            <WarningPills names={p.warnings} />
          </>
        }
      >
        <span className="gl-fact">{rootOf(p.key)}</span>
        <span className="gl-fact">{`last handoff ${readerDate(p.updated, today)}`}</span>
        {members.length > 0 && (
          <span className="gl-fact">
            {`in group ${p.group} with `}
            {members.map((key, i) => (
              <Fragment key={key}>
                {i > 0 && (i === members.length - 1 ? ' and ' : ', ')}
                <Link to={projectPath(key)} go={go} className="gl-link">
                  {label(key, shared)}
                </Link>
              </Fragment>
            ))}
          </span>
        )}
        {p.superseded_by && (
          <span className="gl-fact">
            replaced by{' '}
            {rootOf(p.superseded_by) === '' ? (
              label(p.superseded_by, shared)
            ) : (
              <Link to={projectPath(p.superseded_by)} go={go} className="gl-link">
                {label(p.superseded_by, shared)}
              </Link>
            )}
          </span>
        )}
      </PageHead>
      <div className="gl-cols gl-cols-page">
        <div>
          <section className="gl-nextcard" aria-label="Next action">
            {p.state === 'parked' && <p className="gl-parked">{`Parked: ${p.parked_reason || 'no reason given'}`}</p>}
            <h2 className="gl-eyebrow gl-accent">Next action</h2>
            {s.next ? (
              <div className="gl-prose">
                <Markdown text={s.next} />
              </div>
            ) : (
              <p className="gl-muted">No next action.</p>
            )}
          </section>
          <Block title="Where I stopped" text={s.stopped} empty="No handoff yet." />
          <Block title="Remaining" text={s.remaining} empty="Nothing listed." />
          <Block title="Set aside for later" text={s.ideas} empty="No idea recorded yet." onRail />
        </div>
        <aside className="gl-side">
          {machines.length === 0 ? (
            <section className="gl-card gl-side-card" aria-label="Git">
              <h2 className="gl-eyebrow">Git</h2>
              <p className="gl-muted">No machine has reported this working tree.</p>
            </section>
          ) : (
            machines.map((m) => <GitCard key={m} machine={m} git={p.git[m]!} updated={p.updated} today={today} />)
          )}
          <section className="gl-card gl-side-card" aria-label="Sessions">
            <h2 className="gl-eyebrow">Sessions</h2>
            {sessions.length === 0 && <p className="gl-muted">No session in the last 7 days.</p>}
            {live.length > 0 && <SessionRows sessions={live} today={today} lead={(s) => s.machine} />}
            {ended.length > 0 && live.length === 0 && <p className="gl-muted">No running session.</p>}
            {ended.length > 0 && (
              <Fold label={`Ended (${ended.length})`}>
                {() => <SessionRows sessions={ended} today={today} lead={(s) => s.machine} />}
              </Fold>
            )}
          </section>
          <section className="gl-card gl-side-card" aria-label="Usage">
            <h2 className="gl-eyebrow">Usage</h2>
            <div className="gl-fig">
              <div>
                <b>{figure(p.usage_7)}</b>
                <span>7 days</span>
              </div>
              <div>
                <b>{figure(p.usage_30)}</b>
                <span>30 days</span>
              </div>
              <div>
                <b>{figure(p.usage_all)}</b>
                <span>in total</span>
              </div>
            </div>
            <DayBars days={month} label="Tokens per day, last 30 days" today={today} legend={false} accent />
            <p className="gl-meta gl-card-foot">
              {`${tokens(p.usage_all.cache_read)} cache reads in total${model ? `, mostly ${model}` : ''}`}
            </p>
            {days.length > 0 && (
              <>
                <button type="button" className="gl-fold" aria-expanded={open} aria-controls={tableId} onClick={() => setOpen(!open)}>
                  {open ? <ChevronDown size={14} aria-hidden="true" /> : <ChevronRight size={14} aria-hidden="true" />}
                  {`${FOLD} (${count(days.length, 'row', 'rows')})`}
                </button>
                <div className="gl-scroll" id={tableId} role="region" aria-label={FOLD} tabIndex={0} hidden={!open}>
                  {open && (
                    <table className="gl-table" aria-label={FOLD}>
                      <thead>
                        <tr>
                          <th>Day</th>
                          <th>Model</th>
                          <th className="gl-num">Usage</th>
                          <th className="gl-num">Cache reads</th>
                          <th className="gl-num">Messages</th>
                        </tr>
                      </thead>
                      <tbody>
                        {days.map((d) => (
                          <tr key={`${d.day} ${d.model}`}>
                            <td>{readerDate(d.day, today)}</td>
                            <td>{d.model}</td>
                            <td className="gl-num">{dayFigure(d.usage)}</td>
                            <td className="gl-num">{tokens(d.usage.cache_read)}</td>
                            <td className="gl-num">{d.usage.messages}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  )}
                </div>
              </>
            )}
          </section>
        </aside>
      </div>
    </>
  );
}
