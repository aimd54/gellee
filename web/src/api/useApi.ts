// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

import { useEffect, useState } from 'react';

export type Loaded<T> = { data: T | null; error: string | null; pending: boolean };

type Settled<T> = { data: T | null; error: string | null; path: string };

// useApi fetches /api/v1<path>. A result is returned only for the path
// it was fetched for, so a changed path never shows the previous answer,
// unless keep is set: then the last answer stays in data while the next
// one loads (pending is true) and after a request fails (error is set).
export function useApi<T>(path: string, keep = false): Loaded<T> {
  const [state, setState] = useState<Settled<T>>({ data: null, error: null, path });
  useEffect(() => {
    let live = true;
    fetch(`/api/v1${path}`)
      .then(async (r) => {
        if (!r.ok) throw new Error(`${r.status} ${r.statusText}`);
        return (await r.json()) as T;
      })
      .then(
        (data) => live && setState({ data, error: null, path }),
        (e: unknown) =>
          live &&
          setState((prev) => ({
            data: keep ? prev.data : null,
            error: e instanceof Error ? e.message : String(e),
            path,
          })),
      );
    return () => {
      live = false;
    };
  }, [path, keep]);
  if (state.path === path) return { data: state.data, error: state.error, pending: false };
  return { data: keep ? state.data : null, error: null, pending: true };
}
