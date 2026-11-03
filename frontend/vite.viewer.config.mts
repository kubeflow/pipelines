// Copyright 2026 The Kubeflow Authors. Licensed under the Apache License, Version 2.0.
import { fileURLToPath } from 'node:url';
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  publicDir: false,
  resolve: { alias: { src: fileURLToPath(new URL('./src', import.meta.url)) } },
  plugins: [
    react(),
    {
      name: 'bundle-dagre-dependencies',
      // Dagre's Node/browser guards survive CJS conversion, but `require` does
      // not exist in an IIFE. All these dependencies are installed and bundled.
      transform(code, id) {
        if (/node_modules\/(dagre|graphlib)\/lib\/(graphlib|lodash)\.js$/.test(id)) {
          return code.replace('typeof require === "function"', 'true');
        }
      },
    },
  ],
  define: { 'process.env.NODE_ENV': JSON.stringify('production') },
  build: {
    outDir: fileURLToPath(new URL('../sdk/python/kfp/cli/viewer_assets', import.meta.url)),
    emptyOutDir: true,
    license: { fileName: 'LICENSES.txt' },
    lib: {
      entry: fileURLToPath(new URL('./src/standalone/index.tsx', import.meta.url)),
      name: 'KfpViewer',
      formats: ['iife'],
      fileName: () => 'viewer.js',
      cssFileName: 'viewer',
    },
  },
});
