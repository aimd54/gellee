// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

import { fileURLToPath } from 'node:url';
import tailwindcss from '@tailwindcss/vite';
import react from '@vitejs/plugin-react';
import { defineConfig } from 'vitest/config';

const files = (pkg: string) => fileURLToPath(new URL(`./node_modules/${pkg}/files`, import.meta.url));

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      // The fonts' latin files only, referenced from tokens.css.
      '@fonts-next': files('@fontsource-variable/atkinson-hyperlegible-next'),
      '@fonts-mono': files('@fontsource-variable/atkinson-hyperlegible-mono'),
    },
  },
  build: { outDir: 'dist/app', emptyOutDir: true, assetsInlineLimit: 0 },
  server: { proxy: { '/api': 'http://127.0.0.1:8080' } },
  test: {
    environment: 'jsdom',
    setupFiles: ['src/test/setup.ts'],
    include: ['src/**/*.test.{ts,tsx}'],
    css: false,
  },
});
