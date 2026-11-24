// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

import { vi } from 'vitest';

// stubApi answers fetch from a table of full paths, and 404 for the rest.
export function stubApi(routes: Record<string, unknown>) {
  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: string) => {
      const body = routes[input];
      return body === undefined
        ? new Response('{}', { status: 404, statusText: 'Not Found' })
        : new Response(JSON.stringify(body), { status: 200 });
    }),
  );
}
