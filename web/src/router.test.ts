// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

import { expect, test } from 'vitest';
import { parse } from './router';

test('paths map to views', () => {
  expect(parse('/')).toEqual({ view: 'overview' });
  expect(parse('/projects')).toEqual({ view: 'projects' });
  expect(parse('/projects/alpha/fern')).toEqual({ view: 'project', key: 'alpha/fern' });
  expect(parse('/projects/alpha')).toEqual({ view: 'projects' });
  expect(parse('/usage')).toEqual({ view: 'usage' });
  expect(parse('/ideas')).toEqual({ view: 'ideas' });
  expect(parse('/workers')).toEqual({ view: 'workers' });
  expect(parse('/anything-else')).toEqual({ view: 'overview' });
});

test('encoded segments are decoded into the key', () => {
  expect(parse('/projects/alpha/c%23')).toEqual({ view: 'project', key: 'alpha/c#' });
  expect(parse('/projects/alpha/100%25')).toEqual({ view: 'project', key: 'alpha/100%' });
  expect(parse('/projects/my%20root/my%20tool')).toEqual({ view: 'project', key: 'my root/my tool' });
  expect(parse('/projects/alpha/a%3Fb%26c')).toEqual({ view: 'project', key: 'alpha/a?b&c' });
});

test('an address with a broken escape is read as it stands', () => {
  expect(parse('/projects/alpha/100%')).toEqual({ view: 'project', key: 'alpha/100%' });
});
