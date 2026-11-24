// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

import { render, screen } from '@testing-library/react';
import { expect, test } from 'vitest';
import { Markdown } from './md';

test('paragraphs, lists and inline code become elements', () => {
  const text = 'Paused on `m2-task-17`,\nbefore review.\n\n- one\n  wrapped\n- two';
  const { container } = render(<Markdown text={text} />);
  expect(container.querySelector('p')).toHaveTextContent('Paused on m2-task-17, before review.');
  expect(container.querySelector('code')).toHaveTextContent('m2-task-17');
  expect(screen.getAllByRole('listitem').map((li) => li.textContent)).toEqual(['one wrapped', 'two']);
});

test('markup in the text stays text', () => {
  const { container } = render(<Markdown text={'- <script>alert(1)</script> <b>bold</b>'} />);
  expect(container.querySelector('script')).toBeNull();
  expect(container.querySelector('b')).toBeNull();
  expect(screen.getByRole('listitem')).toHaveTextContent('<script>alert(1)</script> <b>bold</b>');
});

test('empty text renders nothing', () => {
  const { container } = render(<Markdown text="" />);
  expect(container).toBeEmptyDOMElement();
});
