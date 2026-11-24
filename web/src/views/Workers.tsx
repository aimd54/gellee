// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

import type { Worker, WorkersData } from '../api/types';
import { useApi } from '../api/useApi';
import { readerDate, readerMoment } from '../dates';
import { count } from '../format';
import { Fold, Link, PageHead, Pending, Pill, Problems, WarningPills, type Go } from '../parts';
import { label, projectPath, sharedNames } from '../project';
import { SessionRows } from '../sessions';

// seenLine says when a worker last collected: "seen today, 14:00" while
// it is up, "not seen since 9 November" once it is not.
export function seenLine(w: Worker, today: string): string {
  if (w.collected === '') return 'no collection recorded';
  if (w.up) return `seen ${readerMoment(w.collected, today)}`;
  const day = w.collected.slice(0, 10);
  return `not seen since ${day === today ? readerMoment(w.collected, today) : readerDate(day, today)}`;
}

function WorkerCard({ worker: w, today, shared, go }: { worker: Worker; today: string; shared: Set<string>; go: Go }) {
  const live = w.sessions.filter((s) => !s.ended);
  const ended = w.sessions.filter((s) => s.ended);
  const lead = (s: Worker['sessions'][number]) =>
    s.project === 'other' ? (
      <span className="gl-muted">other</span>
    ) : s.tracked ? (
      <Link to={projectPath(s.project)} go={go} className="gl-link">
        {label(s.project, shared)}
      </Link>
    ) : (
      <>
        <span>{label(s.project, shared)}</span> <span className="gl-meta">no status file</span>
      </>
    );
  return (
    <section className="gl-card gl-worker" aria-label={w.name}>
      <header className="gl-worker-head">
        <h2 className="gl-worker-name">{w.name}</h2>
        <span className={w.up ? 'gl-meta' : 'gl-meta gl-late'}>{seenLine(w, today)}</span>
      </header>
      <p className="gl-worker-facts">
        <span>{w.claude_code ? `Claude Code ${w.claude_code}` : 'version unknown'}</span>
        {w.behind && <Pill tone="late">behind</Pill>}
      </p>
      {w.memories && (
        <p className="gl-worker-facts">
          <span>{w.memories.pushed ? `memories pushed ${readerMoment(w.memories.pushed, today)}` : 'memories never pushed'}</span>
          {w.memories.set_aside > 0 && <span>{count(w.memories.set_aside, 'version set aside', 'versions set aside')}</span>}
          {w.memories.stale && <Pill tone="late">sync stale</Pill>}
        </p>
      )}
      {w.sessions.length === 0 && <p className="gl-muted">No session in the last 7 days.</p>}
      {live.length > 0 && <SessionRows sessions={live} today={today} lead={lead} />}
      {ended.length > 0 && (
        <Fold label={`Ended (${ended.length})`}>{() => <SessionRows sessions={ended} today={today} lead={lead} />}</Fold>
      )}
      {w.checkouts.length > 0 && (
        <Fold label={`Checkouts (${w.checkouts.length})`}>
          {() => (
            <ul className="gl-checkouts">
              {w.checkouts.map((c) => (
                <li key={c.project}>
                  <span>{c.project}</span>
                  <span className="gl-mono gl-meta">{c.branch || 'detached'}</span>
                  <WarningPills names={c.warnings} />
                </li>
              ))}
            </ul>
          )}
        </Fold>
      )}
    </section>
  );
}

export function Workers({ go }: { go: Go }) {
  const { data, error } = useApi<WorkersData>('/workers');
  if (!data) return <Pending error={error} what="workers" />;
  const shared = sharedNames(data.workers.flatMap((w) => w.sessions.map((s) => s.project)).filter((k) => k !== 'other'));
  return (
    <>
      <PageHead title="Workers">
        {data.workers.length === 0 ? 'No machine has reported yet.' : "Every figure is from each worker's last collection."}
      </PageHead>
      <Problems items={data.problems} />
      <div className="gl-workers">
        {data.workers.map((w) => (
          <WorkerCard key={w.name} worker={w} today={data.today} shared={shared} go={go} />
        ))}
      </div>
    </>
  );
}
