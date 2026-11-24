// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

export type Theme = 'dark' | 'light';

const KEY = 'gellee-theme';

function stored(): Theme | null {
  try {
    const v = localStorage.getItem(KEY);
    return v === 'dark' || v === 'light' ? v : null;
  } catch {
    return null;
  }
}

export function initTheme(): Theme {
  const theme = stored() ?? (window.matchMedia?.('(prefers-color-scheme: light)').matches ? 'light' : 'dark');
  document.documentElement.dataset.theme = theme;
  return theme;
}

export function toggleTheme(): Theme {
  const theme: Theme = document.documentElement.dataset.theme === 'dark' ? 'light' : 'dark';
  document.documentElement.dataset.theme = theme;
  try {
    localStorage.setItem(KEY, theme);
  } catch {
    // Storage is unavailable; the choice lasts for this page only.
  }
  return theme;
}
