// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

import { useEffect, useState } from 'react';
import type { HealthData, Project, ProjectsData } from '../api/types';
import { useApi } from '../api/useApi';
import { count, figures, tokens } from '../format';
import { PageHead, Pending, Segmented, StateRow, type Go } from '../parts';
import { byHandoff, byUsage, groupId, rootOf, sharedNames, stateLine } from '../project';

const STATES = ['all', 'active', 'parked', 'done'] as const;
type StateChoice = (typeof STATES)[number];
const TITLES: Record<StateChoice, string> = { all: 'All', active: 'Active', parked: 'Parked', done: 'Done' };

export function Projects({ go }: { go: Go }) {
  const { data, error } = useApi<ProjectsData>('/projects');
  const health = useApi<HealthData>('/health');
  const [state, setState] = useState<StateChoice>('all');
  const [root, setRoot] = useState('all');
  const [sort, setSort] = useState('updated');
  const loaded = data !== null && health.data !== null;
  // An address such as /projects#group-fieldwork opens on that group.
  useEffect(() => {
    if (!loaded) return;
    const hash = window.location.hash.slice(1);
    if (!hash) return;
    let target = document.getElementById(hash);
    if (!target) {
      // A hash with a broken escape names nothing: leave the page as it is.
      try {
        target = document.getElementById(decodeURIComponent(hash));
      } catch {
        return;
      }
    }
    target?.scrollIntoView();
  }, [loaded]);
  if (!data || !health.data) return <Pending error={error ?? health.error} what="project list" />;

  const today = health.data.today;
  const all = data.projects;
  const shared = sharedNames(all.map((p) => p.key));
  const roots = [...new Set(all.map((p) => rootOf(p.key)))];
  // Settled over every project, so a filter does not change the unit.
  const figure = figures(all.map((p) => p.usage_30));
  const inRoot = all.filter((p) => root === 'all' || rootOf(p.key) === root);
  const shown = inRoot.filter((p) => state === 'all' || p.state === state);
  const sorted = sort === 'usage' ? byUsage(shown) : byHandoff(shown);

  // A map keeps insertion order, so the groups come out in the order of
  // their best row. Projects without a group are moved to the end.
  const byGroup = new Map<string, Project[]>();
  for (const p of sorted) byGroup.set(p.group, [...(byGroup.get(p.group) ?? []), p]);
  const loose = byGroup.get('');
  byGroup.delete('');
  const sections = [...byGroup.entries()];
  if (loose) sections.push(['', loose]);

  return (
    <>
      <PageHead title="Projects">
        {all.length === 0 ? '0 tracked.' : `${all.length} tracked, across ${count(roots.length, 'root', 'roots')}.`}
      </PageHead>
      <div className="gl-filters">
        <Segmented
          label="State"
          value={state}
          onChange={setState}
          choices={STATES.map((s) => ({
            value: s,
            label: TITLES[s],
            count: inRoot.filter((p) => s === 'all' || p.state === s).length,
          }))}
        />
        <label>
          Root{' '}
          <select value={root} onChange={(e) => setRoot(e.target.value)}>
            <option value="all">all</option>
            {roots.map((r) => (
              <option key={r}>{r}</option>
            ))}
          </select>
        </label>
        <label>
          Sort{' '}
          <select value={sort} onChange={(e) => setSort(e.target.value)}>
            <option value="updated">last handoff</option>
            <option value="usage">usage, 30 days</option>
          </select>
        </label>
      </div>
      {sections.length === 0 && <p className="gl-muted">No project matches these filters.</p>}
      {sections.map(([group, members]) => {
        const id = group ? groupId(group) : 'no-group';
        const held = count(members.length, 'project', 'projects');
        const used = tokens(members.reduce((sum, p) => sum + p.usage_30.tokens, 0));
        return (
          <section key={id} aria-labelledby={id}>
            <div className="gl-grp">
              <h2 id={id}>{group || 'No group'}</h2>
              <span className="gl-meta">{group ? `${held} · ${used} tokens in 30 days` : held}</span>
            </div>
            <ul className="gl-card gl-rows">
              {members.map((p) => (
                <StateRow
                  key={p.key}
                  project={p}
                  today={today}
                  shared={shared}
                  go={go}
                  line={stateLine(p, shared)}
                  figure={
                    <>
                      <b>{figure(p.usage_30)}</b> <span className="gl-under">30 days</span>
                    </>
                  }
                />
              ))}
            </ul>
          </section>
        );
      })}
    </>
  );
}
