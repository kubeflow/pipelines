import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      src: path.resolve(__dirname, 'src'),
    },
  },
  test: {
    environment: 'jsdom',
    setupFiles: ['src/vitest.setup.ts'],
    globals: true,
    css: true,
    include: ['src/**/*.{test,spec}.{ts,tsx}', 'scripts/**/*.{test,spec}.{js,cjs,mjs,ts}'],
    environmentMatchGlobs: [['scripts/**/*.{test,spec}.{js,cjs,mjs,ts}', 'node']],
    server: {
      deps: {
        inline: ['@xyflow/react', '@xyflow/system'],
      },
    },
    coverage: {
      provider: 'v8',
      reporter: ['text', 'lcov', 'json-summary'],
      reportsDirectory: 'coverage/vitest',
      include: ['src/**/*.{ts,tsx,js,jsx}'],
      exclude: [
        'src/**/__snapshots__/**',
        'src/**/__mocks__/**',
        'src/apis/**',
        'src/apisv2beta1/**',
        'src/third_party/**',
        'src/build/**',
      ],
    },
  },
});
