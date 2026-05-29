import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { defineConfig } from 'vitest/config';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

export default defineConfig({
  resolve: {
    alias: {
      src: path.resolve(__dirname, '../src'),
    },
  },
  test: {
    globals: true,
    environment: 'node',
    exclude: ['node_modules', 'dist'],
    pool: 'forks',
    maxWorkers: 1,
    testTimeout: 10000,
  },
});
