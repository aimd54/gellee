// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

import { useState } from 'react';
import type { IdeasData } from '../api/types';
import { useApi } from '../api/useApi';
import { count } from '../format';
import { Inline } from '../md';
import { Link, PageHead, Pending, Segmented, type Go } from '../parts';
import { label, projectPath, sharedNames } from '../project';

export function Ideas({ go }: { go: Go }) {
  const { data, error } = useApi<IdeasData>('/ideas');
  const [only, setOnly] = useState('all');
  if (!data) return <Pending error={error} what="ideas" />;
  const all = data.ideas;
  const keys = [...new Set(all.map((idea) => idea.project))];
  const shared = sharedNames(keys);
  const shown = all.filter((idea) => only === 'all' || idea.project === only);
  return (
    <>
      <PageHead title="Set aside for later">
        {all.length === 0
          ? 'No status file lists an idea yet.'
          : `${count(all.length, 'idea', 'ideas')}, from ${count(keys.length, 'project', 'projects')}.`}
      </PageHead>
      {all.length > 0 && (
        <Segmented
          label="Project"
          value={only}
          onChange={setOnly}
          choices={[
            { value: 'all', label: 'All', count: all.length },
            ...keys.map((key) => ({
              value: key,
              label: label(key, shared),
              count: all.filter((idea) => idea.project === key).length,
            })),
          ]}
        />
      )}
      <ul className="gl-notes">
        {shown.map((idea, i) => (
          <li key={`${idea.project} ${i}`} className="gl-note">
            <p>
              <Inline text={idea.text} />
            </p>
            <Link to={projectPath(idea.project)} go={go} className="gl-meta gl-quiet">
              {label(idea.project, shared)}
            </Link>
          </li>
        ))}
      </ul>
    </>
  );
}
