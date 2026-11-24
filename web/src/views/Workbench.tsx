// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

import { Clock, PencilLine, Upload, type LucideIcon } from 'lucide-react';
import { useState } from 'react';
import type { Project } from '../api/types';
import { attention, type Kind } from '../attention';
import { readerDate } from '../dates';
import { Inline } from '../md';
import { Link, ProjectRow, Segmented, StatePill, StateRow, type Go } from '../parts';
import { byHandoff, label, projectPath, stateLine } from '../project';

type Choice = 'attention' | 'parked' | 'all';

const ICONS: Record<Kind, LucideIcon> = { unpushed: Upload, uncommitted: PencilLine, stale: Clock };

const EMPTY: Record<Choice, string> = {
  attention: 'Nothing needs attention.',
  parked: 'No project is parked.',
  all: 'No project is tracked yet.',
};

// Workbench lists what is not on the bench: what needs attention, what
// is parked, or everything.
export function Workbench({ projects, today, shared, go }: { projects: Project[]; today: string; shared: Set<string>; go: Go }) {
  const items = attention(projects);
  const parked = byHandoff(projects.filter((p) => p.state === 'parked'));
  const all = byHandoff(projects);
  const [choice, setChoice] = useState<Choice>(items.length > 0 ? 'attention' : 'parked');
  const shown = choice === 'attention' ? items.length : choice === 'parked' ? parked.length : all.length;
  return (
    <section className="gl-workbench" aria-labelledby="gl-workbench">
      <div className="gl-sec">
        <h2 id="gl-workbench">On your workbench</h2>
        <Link to="/projects" go={go} className="gl-meta gl-quiet gl-push">
          All projects
        </Link>
      </div>
      <Segmented
        label="Workbench"
        value={choice}
        onChange={setChoice}
        choices={[
          { value: 'attention', label: 'Needs attention', count: items.length },
          { value: 'parked', label: 'Parked', count: parked.length },
          { value: 'all', label: 'All', count: all.length },
        ]}
      />
      {shown === 0 ? (
        <p className="gl-muted">{EMPTY[choice]}</p>
      ) : (
        <ul className="gl-card gl-rows">
          {choice === 'attention' &&
            items.map(({ project: p, kind, sentence }) => {
              const Icon = ICONS[kind];
              return (
                <ProjectRow
                  key={`${p.key} ${kind}`}
                  to={projectPath(p.key)}
                  go={go}
                  projectKey={p.key}
                  icon={<Icon size={18} />}
                  tone="late"
                  name={label(p.key, shared)}
                  line={<Inline text={sentence} />}
                  side={
                    <>
                      <StatePill state={p.state} />
                      <span className="gl-meta">{readerDate(p.updated, today)}</span>
                    </>
                  }
                />
              );
            })}
          {choice === 'parked' &&
            parked.map((p) => <StateRow key={p.key} project={p} today={today} shared={shared} go={go} line={stateLine(p, shared)} />)}
          {choice === 'all' &&
            all.map((p) => (
              <StateRow key={p.key} project={p} today={today} shared={shared} go={go} line={p.next || 'No next action.'} />
            ))}
        </ul>
      )}
    </section>
  );
}
