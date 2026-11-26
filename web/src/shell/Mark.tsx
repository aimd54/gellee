// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

// The mark: a bookmark with a low sun and the horizon cut out of it, drawn
// in the current text colour.
export function Mark({ size = 24 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 32 32" role="img" aria-label="gellee">
      <path d="M8 16V4.5A1.5 1.5 0 0 1 9.5 3H22.5A1.5 1.5 0 0 1 24 4.5V16H21A5 5 0 0 0 11 16Z" fill="currentColor" />
      <path d="M8 18H24V29L16 23L8 29Z" fill="currentColor" />
    </svg>
  );
}
