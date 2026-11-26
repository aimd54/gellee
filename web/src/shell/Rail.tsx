// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

import { ChartColumn, Folder, LayoutGrid, Moon, NotebookPen, Server, Sun } from 'lucide-react';
import { useState } from 'react';
import { Link, type Go } from '../parts';
import { Mark } from './Mark';
import { toggleTheme } from './theme';

const ITEMS = [
  { path: '/', label: 'Overview', id: 'overview', Icon: LayoutGrid },
  { path: '/projects', label: 'Projects', id: 'projects', Icon: Folder },
  { path: '/usage', label: 'Usage', id: 'usage', Icon: ChartColumn },
  { path: '/ideas', label: 'Ideas', id: 'ideas', Icon: NotebookPen },
  { path: '/workers', label: 'Workers', id: 'workers', Icon: Server },
] as const;

export type Section = (typeof ITEMS)[number]['id'];

export function Rail({ current, go, demo = false }: { current: Section; go: Go; demo?: boolean }) {
  const [theme, setTheme] = useState(document.documentElement.dataset.theme ?? 'dark');
  const other = theme === 'dark' ? 'Light theme' : 'Dark theme';
  return (
    <nav className="gl-rail" aria-label="Sections">
      <span className="gl-rail-mark">
        <Mark />
      </span>
      {demo && (
        <span className="gl-rail-demo" role="note" aria-label="Demo: invented data" title="Demo: invented data">
          demo
        </span>
      )}
      {ITEMS.map(({ path, label, id, Icon }) => (
        <Link key={id} to={path} go={go} aria-label={label} title={label} aria-current={current === id ? 'page' : undefined}>
          <Icon size={18} aria-hidden="true" />
        </Link>
      ))}
      <button type="button" className="gl-rail-foot" aria-label={other} title={other} onClick={() => setTheme(toggleTheme())}>
        {theme === 'dark' ? <Sun size={18} aria-hidden="true" /> : <Moon size={18} aria-hidden="true" />}
      </button>
    </nav>
  );
}
