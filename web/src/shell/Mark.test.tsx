// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

import { render, screen } from '@testing-library/react';
import { expect, test } from 'vitest';
import { Mark } from './Mark';

test('the mark draws the bookmark in the current colour', () => {
  render(<Mark size={16} />);
  const svg = screen.getByRole('img', { name: 'gellee' });
  expect(svg).toHaveAttribute('width', '16');
  expect(svg).toHaveAttribute('height', '16');
  expect(svg).toHaveAttribute('viewBox', '0 0 32 32');
  const paths = [...svg.querySelectorAll('path')];
  expect(paths.map((p) => p.getAttribute('d'))).toEqual([
    'M8 16V4.5A1.5 1.5 0 0 1 9.5 3H22.5A1.5 1.5 0 0 1 24 4.5V16H21A5 5 0 0 0 11 16Z',
    'M8 18H24V29L16 23L8 29Z',
  ]);
  for (const p of paths) expect(p).toHaveAttribute('fill', 'currentColor');
});
