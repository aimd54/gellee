// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

import { render, screen, within } from '@testing-library/react';
import { expect, test, vi } from 'vitest';
import { SessionLine, SessionRows } from './sessions';
import { session } from './test/shapes';

test('Open goes to the session in Claude, in a new tab, without a referrer', () => {
  render(<SessionLine session={session({ machine: 'm2', remote: 'session_01Abc', last: '2026-11-11T09:55:00+01:00' })} today="2026-11-11" />);
  expect(screen.getByText('Last session on m2, today, 09:55')).toBeVisible();
  const open = screen.getByRole('link', { name: /^Open\b/ });
  expect(open).toHaveAttribute('href', 'https://claude.ai/code/session_01Abc');
  expect(open).toHaveAttribute('target', '_blank');
  expect(open.getAttribute('rel')?.split(' ')).toContain('noreferrer');
});

test('a session without a remote ID has no Open link', () => {
  render(<SessionLine session={session({ remote: '' })} today="2026-11-11" />);
  expect(screen.queryByRole('link')).toBeNull();
});

test('session rows show the lead, the title, the branch, the last activity and Open', () => {
  render(
    <SessionRows
      sessions={[
        session({ id: 'a', machine: 'm2', title: 'Plan milestone 2', branch: 'main', remote: 'session_01A', last: '2026-11-10T17:40:00+01:00' }),
        session({ id: 'b', machine: 'm1', title: '', branch: '', remote: '', last: '2026-11-09T08:05:00+01:00' }),
      ]}
      today="2026-11-11"
      lead={(s) => s.machine}
    />,
  );
  const rows = screen.getAllByRole('listitem');
  expect(rows).toHaveLength(2);
  expect(rows[0]).toHaveTextContent('m2');
  expect(rows[0]).toHaveTextContent('Plan milestone 2');
  expect(rows[0]).toHaveTextContent('main');
  expect(rows[0]).toHaveTextContent('yesterday, 17:40');
  expect(within(rows[0]!).getByRole('link', { name: /^Open\b/ })).toHaveAttribute('href', 'https://claude.ai/code/session_01A');
  expect(within(rows[0]!).getByRole('link', { name: 'Open Plan milestone 2, in a new tab' })).toBeVisible();
  expect(rows[1]).toHaveTextContent('no title');
  expect(rows[1]).toHaveTextContent('9 November, 08:05');
  expect(within(rows[1]!).queryByRole('link')).toBeNull();
});

test('the Open link of a session without a title is named after the row', () => {
  render(<SessionRows sessions={[session({ id: 'a', title: '', remote: 'session_01A' })]} today="2026-11-11" lead={() => null} />);
  expect(screen.getByRole('link', { name: 'Open untitled session, in a new tab' })).toBeInTheDocument();
});

test('the same session ID on two machines gives two rows with distinct keys', () => {
  const warn = vi.spyOn(console, 'error').mockImplementation(() => {});
  render(
    <SessionRows
      sessions={[session({ id: 'a', machine: 'm1', title: 'One' }), session({ id: 'a', machine: 'm2', title: 'Two' })]}
      today="2026-11-11"
      lead={(s) => s.machine}
    />,
  );
  expect(screen.getAllByRole('listitem')).toHaveLength(2);
  expect(warn).not.toHaveBeenCalled();
  warn.mockRestore();
});

test('the Open link of the last session is named after it', () => {
  render(<SessionLine session={session({ title: 'Plan', remote: 'session_01A' })} today="2026-11-11" />);
  expect(screen.getByRole('link', { name: 'Open Plan, in a new tab' })).toBeInTheDocument();
});
