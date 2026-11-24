// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

import type { ReactNode } from 'react';

function inline(text: string): ReactNode[] {
  return text
    .split(/(`[^`]+`)/)
    .map((part, i) =>
      part.length > 2 && part.startsWith('`') && part.endsWith('`') ? <code key={i}>{part.slice(1, -1)}</code> : part,
    );
}

// Inline renders one line of status text: inline code, nothing else.
export function Inline({ text }: { text: string }) {
  return <>{inline(text)}</>;
}

// Markdown renders the subset status files use: paragraphs, "- " lists
// and inline code. Everything else is shown as the text it is.
export function Markdown({ text }: { text: string }) {
  const blocks: ReactNode[] = [];
  let items: string[] = [];
  let para: string[] = [];
  const flush = () => {
    if (items.length > 0) {
      blocks.push(
        <ul key={blocks.length}>
          {items.map((item, i) => (
            <li key={i}>{inline(item)}</li>
          ))}
        </ul>,
      );
    }
    if (para.length > 0) blocks.push(<p key={blocks.length}>{inline(para.join(' '))}</p>);
    items = [];
    para = [];
  };
  for (const raw of text.split('\n')) {
    const line = raw.trim();
    if (line === '') flush();
    else if (line.startsWith('- ')) {
      if (para.length > 0) flush();
      items.push(line.slice(2));
    } else if (items.length > 0) items[items.length - 1] += ` ${line}`;
    else para.push(line);
  }
  flush();
  return <>{blocks}</>;
}
