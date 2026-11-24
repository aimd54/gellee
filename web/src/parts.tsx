// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

import { ChevronDown, ChevronRight, Folder } from 'lucide-react';
import { useContext, useId, useLayoutEffect, useRef, useState, type ComponentProps, type ReactNode } from 'react';
import type { Problem, Project } from './api/types';
import { readerDate } from './dates';
import { Navigation } from './navigation';
import { Inline } from './md';
import { label, projectPath } from './project';

export type Go = (path: string) => void;
export type Tone = 'accent' | 'unknown' | 'done' | 'late' | 'failed';

type LinkProps = Omit<ComponentProps<'a'>, 'href' | 'onClick'> & { to: string; go: Go };

// Link is a link inside the interface: a plain click is routed without
// loading a page, a modified click is left to the browser.
export function Link({ to, go, ...rest }: LinkProps) {
  return (
    <a
      href={to}
      {...rest}
      onClick={(e) => {
        if (e.button !== 0 || e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return;
        e.preventDefault();
        go(to);
      }}
    />
  );
}

// PageHead is the top of a view: an optional small line, the heading,
// whatever sits beside it, and one sentence under it.
export function PageHead({
  eyebrow,
  title,
  after,
  children,
}: {
  eyebrow?: ReactNode;
  title: ReactNode;
  after?: ReactNode;
  children?: ReactNode;
}) {
  const heading = useRef<HTMLHeadingElement>(null);
  const navigation = useContext(Navigation);
  // Before paint, so focus never rests on the view between its first
  // frame and the heading's.
  useLayoutEffect(() => {
    if (navigation.moved) heading.current?.focus({ preventScroll: true });
  }, [navigation]);
  return (
    <header className="gl-head">
      {eyebrow && <div className="gl-eyebrow">{eyebrow}</div>}
      <div className="gl-head-title">
        <h1 className="gl-display" ref={heading} tabIndex={-1}>
          {title}
        </h1>
        {after}
      </div>
      {children && <div className="gl-lede">{children}</div>}
    </header>
  );
}

export function Pill({ tone, children }: { tone: Tone; children: ReactNode }) {
  return <span className={`gl-pill gl-tone-${tone}`}>{children}</span>;
}

const STATE_TONE: Record<string, Tone> = { active: 'accent', parked: 'unknown', done: 'done' };

export function StatePill({ state }: { state: string }) {
  return <Pill tone={STATE_TONE[state] ?? 'unknown'}>{state}</Pill>;
}

const WARNINGS: Record<string, string> = {
  stale: 'stale status',
  unpushed: 'unpushed work',
  uncommitted: 'uncommitted work',
};

export function WarningPills({ names }: { names: string[] }) {
  return (
    <>
      {names.map((n) => (
        <Pill key={n} tone="late">
          {WARNINGS[n] ?? n}
        </Pill>
      ))}
    </>
  );
}

export function Segmented<T extends string>({
  label: name,
  choices,
  value,
  onChange,
}: {
  label: string;
  choices: { value: T; label: string; count?: number }[];
  value: T;
  onChange: (value: T) => void;
}) {
  return (
    <div className="gl-seg" role="group" aria-label={name}>
      {choices.map((c) => (
        <button key={c.value} type="button" aria-pressed={c.value === value} onClick={() => onChange(c.value)}>
          {c.label}
          {c.count !== undefined && (
            <>
              {' '}
              <small>{c.count}</small>
            </>
          )}
        </button>
      ))}
    </div>
  );
}

// ProjectRow is one project on one line. The whole row is one link.
export function ProjectRow({
  to,
  go,
  projectKey,
  icon,
  tone,
  name,
  line,
  figure: amount,
  side,
}: {
  to: string;
  go: Go;
  projectKey: string;
  icon: ReactNode;
  tone: Tone;
  name: string;
  line: ReactNode;
  figure?: ReactNode;
  side: ReactNode;
}) {
  return (
    <li>
      <Link to={to} go={go} className="gl-row" data-project={projectKey}>
        <span className={`gl-tile gl-tone-${tone}`} aria-hidden="true">
          {icon}
        </span>
        <span className="gl-row-text">
          <b>{name}</b>
          <span className="gl-row-line">{line}</span>
        </span>
        {amount && <span className="gl-row-figure">{amount}</span>}
        <span className="gl-row-side">{side}</span>
        <ChevronRight size={18} className="gl-muted" aria-hidden="true" />
      </Link>
    </li>
  );
}

// StateRow is the row of a project wherever it is listed by state: a
// folder, the state, and its warnings or else the date of its handoff.
export function StateRow({
  project: p,
  today,
  shared,
  go,
  line,
  figure: amount,
}: {
  project: Project;
  today: string;
  shared: Set<string>;
  go: Go;
  line: string;
  figure?: ReactNode;
}) {
  return (
    <ProjectRow
      to={projectPath(p.key)}
      go={go}
      projectKey={p.key}
      icon={<Folder size={18} />}
      tone={p.state === 'active' ? 'accent' : 'unknown'}
      name={label(p.key, shared)}
      line={<Inline text={line} />}
      figure={amount}
      side={
        <>
          <StatePill state={p.state} />
          {p.warnings.length > 0 ? (
            <span className="gl-row-warns">
              <WarningPills names={p.warnings} />
            </span>
          ) : (
            <span className="gl-meta">{readerDate(p.updated, today)}</span>
          )}
        </>
      }
    />
  );
}

export function Problems({ items }: { items: Problem[] }) {
  if (items.length === 0) return null;
  return (
    <section className="gl-problems" role="alert">
      <b>{items.length === 1 ? '1 file could not be read' : `${items.length} files could not be read`}</b>
      <ul>
        {items.map((p) => (
          <li key={p.file}>
            <span className="gl-mono">{p.file}</span>: {p.message}
          </li>
        ))}
      </ul>
    </section>
  );
}

export function Pending({ error, what }: { error: string | null; what: string }) {
  return error ? (
    <p role="alert">
      The {what} could not be loaded: {error}
    </p>
  ) : (
    <p className="gl-muted">Loading</p>
  );
}

// Fold shows its label as a button and renders its content only while
// open.
export function Fold({ label, children }: { label: string; children: () => ReactNode }) {
  const [open, setOpen] = useState(false);
  const id = useId();
  return (
    <>
      <button type="button" className="gl-fold" aria-expanded={open} aria-controls={id} onClick={() => setOpen(!open)}>
        {open ? <ChevronDown size={14} aria-hidden="true" /> : <ChevronRight size={14} aria-hidden="true" />}
        {label}
      </button>
      <div id={id} hidden={!open}>
        {open && children()}
      </div>
    </>
  );
}
