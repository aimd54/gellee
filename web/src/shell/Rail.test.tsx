// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

import { fireEvent, render, screen, within } from '@testing-library/react';
import { afterEach, expect, test, vi } from 'vitest';
import { Rail } from './Rail';

afterEach(() => localStorage.clear());

test('the rail links every section by name and marks the current one', () => {
  document.documentElement.dataset.theme = 'dark';
  const go = vi.fn();
  render(<Rail current="projects" go={go} />);
  const nav = screen.getByRole('navigation', { name: 'Sections' });
  expect(
    within(nav)
      .getAllByRole('link')
      .map((a) => [a.getAttribute('aria-label'), a.getAttribute('title'), a.getAttribute('href')]),
  ).toEqual([
    ['Overview', 'Overview', '/'],
    ['Projects', 'Projects', '/projects'],
    ['Usage', 'Usage', '/usage'],
    ['Ideas', 'Ideas', '/ideas'],
    ['Workers', 'Workers', '/workers'],
  ]);
  expect(within(nav).getByRole('link', { name: 'Projects' })).toHaveAttribute('aria-current', 'page');
  expect(within(nav).getByRole('link', { name: 'Usage' })).not.toHaveAttribute('aria-current');
  expect(within(nav).getByRole('img', { name: 'gellee' })).toBeVisible();
  fireEvent.click(within(nav).getByRole('link', { name: 'Ideas' }));
  expect(go).toHaveBeenCalledWith('/ideas');
});

test('the theme switch names the theme it switches to', () => {
  document.documentElement.dataset.theme = 'dark';
  render(<Rail current="overview" go={vi.fn()} />);
  fireEvent.click(screen.getByRole('button', { name: 'Light theme' }));
  expect(document.documentElement.dataset.theme).toBe('light');
  expect(screen.getByRole('button', { name: 'Dark theme' })).toHaveAttribute('title', 'Dark theme');
});
