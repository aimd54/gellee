// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

import { useCallback, useEffect, useState } from 'react';

export type Route =
  | { view: 'overview' }
  | { view: 'projects' }
  | { view: 'project'; key: string }
  | { view: 'usage' }
  | { view: 'ideas' }
  | { view: 'workers' };

// decode reads one path segment; one with a broken escape stands as written.
function decode(segment: string): string {
  try {
    return decodeURIComponent(segment);
  } catch {
    return segment;
  }
}

export function parse(path: string): Route {
  const parts = path.split('/').filter(Boolean).map(decode);
  if (parts[0] === 'projects') return parts.length === 3 ? { view: 'project', key: `${parts[1]}/${parts[2]}` } : { view: 'projects' };
  if (parts[0] === 'usage') return { view: 'usage' };
  if (parts[0] === 'ideas') return { view: 'ideas' };
  if (parts[0] === 'workers') return { view: 'workers' };
  return { view: 'overview' };
}

export function useRoute(): [Route, (path: string) => void] {
  const [path, setPath] = useState(window.location.pathname);
  useEffect(() => {
    const onPop = () => setPath(window.location.pathname);
    window.addEventListener('popstate', onPop);
    return () => window.removeEventListener('popstate', onPop);
  }, []);
  const go = useCallback((to: string) => {
    window.history.pushState(null, '', to);
    // The address may carry a hash or a query; the route is the path.
    setPath(window.location.pathname);
  }, []);
  return [parse(path), go];
}
