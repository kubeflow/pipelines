import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import { visualizer } from 'rollup-plugin-visualizer';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const proxyTarget = 'http://localhost:3001';
const proxyPaths = [
  '/api',
  '/apis',
  '/apps',
  '/artifacts',
  '/hub',
  '/k8s',
  '/ml_metadata',
  '/system',
  '/visualizations',
];

const proxy = proxyPaths.reduce<Record<string, { target: string; changeOrigin: boolean }>>(
  (acc, prefix) => {
    acc[prefix] = { target: proxyTarget, changeOrigin: true };
    return acc;
  },
  {},
);

export default defineConfig(({ mode }) => ({
  base: './',
  plugins: [
    react(),
    mode === 'analyze' &&
      visualizer({
        filename: 'build/bundle-report.html',
        gzipSize: true,
        brotliSize: true,
        open: true,
      }),
  ].filter(Boolean),
  resolve: {
    alias: {
      src: path.resolve(__dirname, 'src'),
      'react-virtualized': path.resolve(
        __dirname,
        'node_modules/react-virtualized/dist/commonjs/index.js',
      ),
    },
  },
  server: {
    port: 3000,
    proxy,
  },
  build: {
    target: 'es2015',
    outDir: 'build',
    assetsDir: 'static',
    sourcemap: true,
    commonjsOptions: {
      include: [/node_modules/, /src\/generated/, /src\/third_party\/mlmd\/generated/],
    },
  },
}));
