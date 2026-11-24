// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

import { fireEvent, render, screen, within } from '@testing-library/react';
import { expect, test, vi } from 'vitest';
import { stubApi } from '../test/api';
import { Ideas } from './Ideas';

const ideas = {
  ideas: [
    { project: 'alpha/fern', text: 'Compare two mixes' },
    { project: 'alpha/fern', text: 'Write up the `sampling` measurements' },
    { project: 'beta/tool', text: 'A table parser' },
  ],
};

const notes = () => screen.getAllByRole('listitem').map((li) => li.textContent);

test('ideas are notes with their project, filtered by project', async () => {
  stubApi({ '/api/v1/ideas': ideas });
  const go = vi.fn();
  render(<Ideas go={go} />);
  expect(await screen.findByRole('heading', { level: 1 })).toHaveTextContent('Set aside for later');
  expect(screen.getByText('3 ideas, from 2 projects.')).toBeVisible();
  expect(notes()).toEqual(['Compare two mixesfern', 'Write up the sampling measurementsfern', 'A table parsertool']);
  expect(screen.getByText('sampling').tagName).toBe('CODE');

  const filter = within(screen.getByRole('group', { name: 'Project' }));
  expect(filter.getAllByRole('button').map((b) => b.textContent)).toEqual(['All 3', 'fern 2', 'tool 1']);
  expect(filter.getByRole('button', { name: 'All 3' })).toHaveAttribute('aria-pressed', 'true');
  fireEvent.click(filter.getByRole('button', { name: 'tool 1' }));
  expect(notes()).toEqual(['A table parsertool']);
  expect(filter.getByRole('button', { name: 'tool 1' })).toHaveAttribute('aria-pressed', 'true');

  const link = screen.getByRole('link', { name: 'tool' });
  expect(link).toHaveAttribute('href', '/projects/beta/tool');
  fireEvent.click(link);
  expect(go).toHaveBeenCalledWith('/projects/beta/tool');
});

test('one idea is written in the singular', async () => {
  stubApi({ '/api/v1/ideas': { ideas: ideas.ideas.slice(2) } });
  render(<Ideas go={vi.fn()} />);
  expect(await screen.findByText('1 idea, from 1 project.')).toBeVisible();
});

test('without any idea the page says so and shows no filter', async () => {
  stubApi({ '/api/v1/ideas': { ideas: [] } });
  render(<Ideas go={vi.fn()} />);
  expect(await screen.findByText('No status file lists an idea yet.')).toBeVisible();
  expect(screen.queryByRole('group')).toBeNull();
  expect(screen.queryAllByRole('listitem')).toHaveLength(0);
});

test('a failed request says so', async () => {
  stubApi({});
  render(<Ideas go={vi.fn()} />);
  expect(await screen.findByRole('alert')).toHaveTextContent('The ideas could not be loaded: 404 Not Found');
});

test('the project under a note is underlined at rest', async () => {
  stubApi({ '/api/v1/ideas': { ideas: [{ project: 'alpha/fern', text: 'One.' }] } });
  render(<Ideas go={vi.fn()} />);
  expect(await screen.findByRole('link', { name: 'fern' })).toHaveClass('gl-quiet');
});
