// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

import { ChartColumn, NotebookPen } from 'lucide-react';
import type { IdeasData, OverviewData, ProjectsData, UsageData } from '../api/types';
import { useApi } from '../api/useApi';
import { collectedLine, longDay } from '../dates';
import { amount, count, figure, tokens } from '../format';
import { Inline } from '../md';
import { Link, PageHead, Pending, Pill, Problems, type Go } from '../parts';
import { byHandoff, label, sharedNames } from '../project';
import { Bench } from './Bench';
import { DayBars } from './DayBars';
import { Workbench } from './Workbench';

function Stat({ value, label: text, late }: { value: string | number; label: string; late?: boolean }) {
  return (
    <div className="gl-stat">
      <b className={late ? 'gl-late' : undefined}>{value}</b>
      <span>{text}</span>
    </div>
  );
}

export function Overview({ go }: { go: Go }) {
  const overview = useApi<OverviewData>('/overview');
  const projects = useApi<ProjectsData>('/projects');
  const ideas = useApi<IdeasData>('/ideas');
  // The default period of /usage is the last 30 days, the period of the
  // chart, and its total counts work outside the tracked projects too.
  const month = useApi<UsageData>('/usage');
  if (!overview.data || !projects.data || !ideas.data || !month.data) {
    return <Pending error={overview.error ?? projects.error ?? ideas.error ?? month.error} what="overview" />;
  }
  const o = overview.data;
  const all = projects.data.projects;
  const total = month.data.total;
  const later = ideas.data.ideas;
  const shared = sharedNames(all.map((p) => p.key));
  const active = byHandoff(all.filter((p) => p.state === 'active'));
  const parked = all.filter((p) => p.state === 'parked').length;
  const done = all.filter((p) => p.state === 'done').length;
  const needing = all.filter((p) => p.warnings.length > 0).length;
  const over = active.length - o.limit;
  const bench = active.length === 0 ? 'No project' : count(active.length, 'project', 'projects');
  return (
    <>
      <PageHead eyebrow={`${longDay(o.today)} · ${collectedLine(o.collected, o.today)}`} title="Pick up the thread.">
        <span>{`${bench} on the bench, the limit is ${o.limit}.`}</span>
        {over > 0 && <Pill tone="late">{over} over the limit</Pill>}
      </PageHead>
      <Problems items={o.problems} />
      <Bench projects={active} today={o.today} shared={shared} go={go} />
      <div className="gl-stats">
        <Stat value={all.length} label="projects tracked" />
        <Stat value={parked} label={`parked, ${done} done`} />
        <Stat value={needing} label={needing === 1 ? 'needs attention' : 'need attention'} late={needing > 0} />
        <Stat value={figure(o.week)} label={o.week.cost === null ? 'tokens in 7 days' : 'in 7 days'} />
      </div>
      <div className="gl-cols">
        <Workbench projects={all} today={o.today} shared={shared} go={go} />
        <aside className="gl-side">
          <section className="gl-card gl-side-card" aria-labelledby="gl-month">
            <h2 id="gl-month" className="gl-card-title">
              <ChartColumn size={18} aria-hidden="true" />
              Last 30 days
            </h2>
            <DayBars days={o.days} label="Tokens per day, by project" today={o.today} />
            <p className="gl-meta gl-card-foot">
              <span>{`${amount(total)}, ${tokens(total.cache_read)} cache reads`}</span>
              <Link to="/usage" go={go} className="gl-link gl-push">
                Usage in detail
              </Link>
            </p>
          </section>
          {later.length > 0 && (
            <section className="gl-card gl-side-card gl-on-rail" aria-labelledby="gl-later">
              <h2 id="gl-later" className="gl-card-title">
                <NotebookPen size={18} aria-hidden="true" />
                Set aside for later
              </h2>
              <ul className="gl-items">
                {later.slice(0, 3).map((idea, i) => (
                  <li key={i}>
                    <b>
                      <Inline text={idea.text} />
                    </b>
                    <span className="gl-meta">{label(idea.project, shared)}</span>
                  </li>
                ))}
              </ul>
              <Link to="/ideas" go={go} className="gl-link">
                All ideas
              </Link>
            </section>
          )}
        </aside>
      </div>
    </>
  );
}
