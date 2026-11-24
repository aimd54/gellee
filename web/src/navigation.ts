// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

import { createContext } from 'react';

// Navigation tells a view's heading whether the person just moved here
// from another view: then the heading takes focus, so the change is
// announced and the next Tab starts from the new content. On the first
// load focus is left alone.
export const Navigation = createContext<{ moved: boolean }>({ moved: false });
