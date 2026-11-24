// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

import { ExternalLink } from 'lucide-react';
import type { ReactNode } from 'react';
import type { MachineSession, Session } from './api/types';
import { readerMoment } from './dates';

// sessionAddress is where a session connected through Remote Control
// opens in Claude.
export function sessionAddress(remote: string): string {
  return `https://claude.ai/code/${encodeURIComponent(remote)}`;
}

// OpenSession opens a session in a new tab that is not told which page
// sent it. The visible text is Open; the accessible name adds which
// session.
export function OpenSession({ remote, name }: { remote: string; name: string }) {
  return (
    <a className="gl-open" href={sessionAddress(remote)} target="_blank" rel="noreferrer">
      Open{' '}
      <span className="gl-sr-only">{`${name}, in a new tab`}</span>
      <ExternalLink size={12} aria-hidden="true" />
    </a>
  );
}

// SessionRows lists sessions in the order given: what leads each row
// (its project or its machine), its title, branch and last activity,
// and Open when it has a remote ID.
export function SessionRows<T extends Session>({
  sessions,
  today,
  lead,
}: {
  sessions: T[];
  today: string;
  lead: (s: T) => ReactNode;
}) {
  return (
    <ul className="gl-sessions">
      {sessions.map((s) => (
        <li key={'machine' in s ? `${String(s.machine)}/${s.id}` : s.id} className="gl-session">
          <span className="gl-session-lead">{lead(s)}</span>
          <span className={s.title ? 'gl-session-title' : 'gl-session-title gl-muted'}>{s.title || 'no title'}</span>
          {s.branch && (
            <span className="gl-mono gl-meta gl-session-branch" title={s.branch}>
              {s.branch}
            </span>
          )}
          <span className="gl-meta">{readerMoment(s.last, today)}</span>
          {s.remote && <OpenSession remote={s.remote} name={s.title || 'untitled session'} />}
        </li>
      ))}
    </ul>
  );
}

// SessionLine is the last session of a bench card's project.
export function SessionLine({ session: s, today }: { session: MachineSession; today: string }) {
  return (
    <p className="gl-hero-session">
      <span>{`Last session on ${s.machine}, ${readerMoment(s.last, today)}`}</span>
      {s.remote && <OpenSession remote={s.remote} name={s.title || 'the last session'} />}
    </p>
  );
}
