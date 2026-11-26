// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

import { fireEvent, render, screen } from '@testing-library/react';
import { afterEach, expect, test } from 'vitest';
import { App } from './App';
import { stubApi } from './test/api';
import { project } from './test/shapes';

const fern = {
  project: { ...project({ key: 'alpha/fern' }), sections: { stopped: '', next: '', remaining: '', ideas: '' } },
  group_members: [],
  days: [],
  sessions: [],
};

function stub() {
  stubApi({
    '/api/v1/ideas': { ideas: [{ project: 'alpha/fern', text: 'One idea.' }] },
    '/api/v1/projects/alpha/fern': fern,
    '/api/v1/health': { today: '2026-11-11' },
  });
}

// focusOnArrival resolves to the element that has focus when an h1 with
// this text first appears: a mutation observer runs right after React
// commits, before the effects it runs after paint, so focus left for those
// effects to move is seen where it rests in between.
function focusOnArrival(text: string): Promise<Element | null> {
  return new Promise((resolve) => {
    const observer = new MutationObserver(() => {
      if ([...document.querySelectorAll('h1')].some((h) => h.textContent?.trim() === text)) {
        observer.disconnect();
        resolve(document.activeElement);
      }
    });
    observer.observe(document.body, { childList: true, subtree: true, characterData: true });
  });
}

afterEach(() => {
  window.history.replaceState(null, '', '/');
  document.title = 'gellee';
});

test('a route change moves focus to the new heading and names the page', async () => {
  window.history.replaceState(null, '', '/ideas');
  stub();
  render(<App />);
  const first = await screen.findByRole('heading', { level: 1 });
  expect(document.title).toBe('Ideas | gellee');
  // The first load leaves focus alone.
  expect(first).not.toHaveFocus();
  expect(document.body).toHaveFocus();

  const fernFocus = focusOnArrival('fern');
  fireEvent.click(await screen.findByRole('link', { name: 'fern' }));
  const heading = await screen.findByRole('heading', { level: 1, name: 'fern' });
  expect(await fernFocus).toBe(heading);
  expect(heading).toHaveFocus();
  expect(document.title).toBe('fern | gellee');
  expect(screen.getByRole('main')).toHaveAttribute('tabindex', '-1');

  const ideasFocus = focusOnArrival('Set aside for later');
  fireEvent.click(screen.getByRole('link', { name: 'Ideas' }));
  const ideas = await screen.findByRole('heading', { level: 1, name: 'Set aside for later' });
  expect(await ideasFocus).toBe(ideas);
  expect(ideas).toHaveFocus();
  expect(document.title).toBe('Ideas | gellee');
});

test('the rail shows the demo badge when the server says it is a demo', async () => {
  stubApi({ '/api/v1/health': { today: '2026-11-11', demo: true }, '/api/v1/ideas': { ideas: [] } });
  window.history.pushState({}, '', '/ideas');
  render(<App />);
  expect(await screen.findByRole('note', { name: 'Demo: invented data' })).toBeVisible();
});
