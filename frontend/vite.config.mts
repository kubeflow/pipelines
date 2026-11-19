// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

const directory = path.dirname(fileURLToPath(import.meta.url));
// Development uses the deployed authenticated proxy, never a KFP API server.
const target = process.env.KFP_NATIVE_PROXY || 'http://127.0.0.1:3003';
export default defineConfig({
  base: './',
  plugins: [react()],
  resolve: { alias: { src: path.resolve(directory, 'src') } },
  server: {
    host: '127.0.0.1',
    port: 3000,
    proxy: {
      '^/argo/': { target, changeOrigin: true },
      '^/kubernetes/': { target, changeOrigin: true },
    },
  },
  build: {
    outDir: 'build',
    assetsDir: 'static',
    sourcemap: true,
    rollupOptions: {
      input: {
        main: path.resolve(directory, 'index.html'),
        argo: path.resolve(directory, 'argo.html'),
      },
    },
  },
});
