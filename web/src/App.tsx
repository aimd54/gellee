// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

import { useLayoutEffect, useRef } from 'react';
import { Navigation } from './navigation';
import { nameOf } from './project';
import { useRoute, type Route } from './router';
import { Rail } from './shell/Rail';
import { Ideas } from './views/Ideas';
import { Workers } from './views/Workers';
import { Overview } from './views/Overview';
import { ProjectPage } from './views/ProjectPage';
import { Projects } from './views/Projects';
import { UsageView } from './views/UsageView';

const NAMES = { overview: 'Overview', projects: 'Projects', usage: 'Usage', ideas: 'Ideas', workers: 'Workers' } as const;

function titleOf(route: Route): string {
  return `${route.view === 'project' ? nameOf(route.key) : NAMES[route.view]} | gellee`;
}

export function App() {
  const [route, go] = useRoute();
  // A new view starts at its top: the key gives each one its own
  // scrolling element.
  const page = route.view === 'project' ? route.key : route.view;
  const main = useRef<HTMLElement>(null);
  const seen = useRef<string | null>(null);
  const moved = useRef(false);
  useLayoutEffect(() => {
    document.title = titleOf(route);
    if (seen.current !== null && seen.current !== page) {
      // The link that was activated is gone with the old view: focus
      // goes to the new one's heading. A heading drawn in this commit ran
      // its own effect before this one, while nothing had moved yet, so it
      // is focused here; one drawn later focuses itself.
      moved.current = true;
      const heading = main.current?.querySelector<HTMLElement>('h1');
      (heading ?? main.current)?.focus({ preventScroll: true });
    }
    seen.current = page;
  });
  return (
    <div className="gl-app">
      <Rail current={route.view === 'project' ? 'projects' : route.view} go={go} />
      <main className="gl-main" key={page} ref={main} tabIndex={-1}>
        <Navigation.Provider value={{ get moved() { return moved.current; } }}>
          {route.view === 'overview' && <Overview go={go} />}
          {route.view === 'projects' && <Projects go={go} />}
          {route.view === 'project' && <ProjectPage projectKey={route.key} go={go} />}
          {route.view === 'usage' && <UsageView go={go} />}
          {route.view === 'ideas' && <Ideas go={go} />}
          {route.view === 'workers' && <Workers go={go} />}
        </Navigation.Provider>
      </main>
    </div>
  );
}
