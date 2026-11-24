// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

import { ArrowRight } from 'lucide-react';
import type { Project } from '../api/types';
import { readerDate } from '../dates';
import { figures } from '../format';
import { Inline } from '../md';
import { Link, StatePill, WarningPills, type Go } from '../parts';
import { label, projectPath, rootOf } from '../project';
import { SessionLine } from '../sessions';

// Bench shows one card per active project. The first is where work
// stopped last, and says so.
export function Bench({ projects, today, shared, go }: { projects: Project[]; today: string; shared: Set<string>; go: Go }) {
  if (projects.length === 0) {
    return (
      <p className="gl-muted">
        Nothing on the bench. Activate a project with <code>/handoff activate</code>.
      </p>
    );
  }
  const figure = figures(projects.map((p) => p.usage_7));
  return (
    <section className="gl-bench" aria-label="The bench">
      {projects.map((p, i) => (
        <div key={p.key} className="gl-hero-wrap">
          <Link to={projectPath(p.key)} go={go} className={i === 0 ? 'gl-hero gl-hero-first' : 'gl-hero'} data-project={p.key}>
            <span className="gl-hero-head">
              <StatePill state={p.state} />
              {i === 0 && <span className="gl-eyebrow gl-accent">Where you left off</span>}
              <span className="gl-meta gl-push">{readerDate(p.updated, today)}</span>
            </span>
            <h2 className="gl-hero-name">{label(p.key, shared)}</h2>
            <p className="gl-hero-where">
              {i === 0 ? <Inline text={p.stopped || 'No handoff yet.'} /> : p.group ? `${p.group} group` : rootOf(p.key)}
            </p>
            <span className="gl-next">
              <ArrowRight size={18} aria-hidden="true" />
              <span>
                <Inline text={p.next || 'No next action.'} />
              </span>
            </span>
            <span className="gl-hero-foot">
              <WarningPills names={p.warnings} />
              <span className="gl-meta gl-push">
                <b>{figure(p.usage_7)}</b> in 7 days
              </span>
            </span>
          </Link>
          {p.last_session && <SessionLine session={p.last_session} today={today} />}
        </div>
      ))}
    </section>
  );
}
