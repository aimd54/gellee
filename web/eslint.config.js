// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

import js from '@eslint/js';
import reactHooks from 'eslint-plugin-react-hooks';
import globals from 'globals';
import tseslint from 'typescript-eslint';

export default tseslint.config(
  { ignores: ['dist', 'node_modules', 'test-results', 'playwright-report'] },
  js.configs.recommended,
  ...tseslint.configs.strict,
  reactHooks.configs.flat['recommended-latest'],
  { languageOptions: { globals: { ...globals.browser, ...globals.node } } },
  // noUncheckedIndexedAccess types every indexed read as possibly undefined;
  // a non-null assertion is how the code says it has already checked.
  { rules: { '@typescript-eslint/no-non-null-assertion': 'off' } },
);
