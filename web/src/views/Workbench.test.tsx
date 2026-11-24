// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

import { render, screen } from '@testing-library/react';
import { expect, test, vi } from 'vitest';
import { project } from '../test/shapes';
import { Workbench } from './Workbench';

test('the link to all projects is underlined at rest', () => {
  render(<Workbench projects={[project({})]} today="2026-11-11" shared={new Set()} go={vi.fn()} />);
  expect(screen.getByRole('link', { name: 'All projects' })).toHaveClass('gl-quiet');
});
