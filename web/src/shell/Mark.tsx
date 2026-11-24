// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

// The mark: a leaf, drawn in the current text colour.
export function Mark({ size = 24 }: { size?: number }) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      role="img"
      aria-label="gellee"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.7"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <path d="M5 19c0-9 5-14 15-14 0 10-5 15-14 15" />
      <path d="M5 19c3-5 6-8 10-10" />
    </svg>
  );
}
