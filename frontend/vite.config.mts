import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { defineConfig, type Plugin } from 'vite';
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

/**
 * Vite plugin to transform CommonJS protobuf files (in src/generated/ and
 * src/third_party/mlmd/generated/) into ES modules so they work with Vite's
 * dev server which serves source files as native ESM.
 */
function cjsProtobufPlugin(): Plugin {
  return {
    name: 'cjs-protobuf-shim',
    enforce: 'pre',
    transform(code: string, id: string) {
      if (!id.endsWith('.js')) return null;
      if (!id.includes('/generated/')) return null;
      if (!code.includes('require(')) return null;

      // Collect require() calls and convert to ESM imports
      const imports: string[] = [];
      let importCounter = 0;
      let transformed = code;

      // Handle ALL require() patterns in one pass:
      //   var X = require('...');
      //   X.Y = require('...');
      //   const X = {}; ... X.Y = require('...');
      transformed = transformed.replace(
        /(var\s+(\w+)\s*=\s*)?(?:(\w[\w.]*)\s*=\s*)?require\(\s*['"]([^'"]+)['"]\s*\);?/g,
        (fullMatch, varDecl, varName, dotTarget, modPath) => {
          const tempVar = `__cjs_import_${importCounter++}`;
          imports.push(`import * as ${tempVar} from '${modPath}';`);
          // Spread into a mutable copy so properties can be added later
          const mutableCopy = `Object.assign({}, ${tempVar})`;
          if (varDecl && varName) {
            // var X = require('...')
            return `var ${varName} = ${mutableCopy};`;
          } else if (dotTarget) {
            // X.Y = require('...')
            return `${dotTarget} = ${mutableCopy};`;
          }
          return fullMatch; // fallback, shouldn't happen
        },
      );

      // Extract top-level export names from goog.exportSymbol() calls
      // Match patterns like 'proto.namespace.ExportName' (exactly 3 dot-separated parts)
      const exportNames = new Set<string>();
      const symbolRegex = /goog\.exportSymbol\(\s*['"]proto\.\w+\.(\w+)['"]/g;
      let match;
      while ((match = symbolRegex.exec(code)) !== null) {
        exportNames.add(match[1]);
      }

      // Build the transformed module
      const preamble = [
        ...imports,
        'var exports = {};',
        'var module = { exports: exports };',
        '',
      ].join('\n');

      // Build named exports from either goog.object.extend(exports, ...) or module.exports = ...
      const namedExportLines: string[] = [];
      if (exportNames.size > 0) {
        for (const name of exportNames) {
          namedExportLines.push(`export var ${name} = exports.${name};`);
        }
      }

      // Handle module.exports = X pattern (grpc_web_pb uses this)
      // Extract export names from the module.exports assignment target
      const moduleExportsMatch = code.match(/module\.exports\s*=\s*([\w.]+);/);
      if (moduleExportsMatch) {
        const ns = moduleExportsMatch[1]; // e.g., proto.ml_metadata
        // For grpc_web, we know the exports from the file structure
        // Read class names defined in the file
        const classRegex = /^(proto\.[\w.]+\.(\w+Client))\s*=/gm;
        let classMatch;
        while ((classMatch = classRegex.exec(code)) !== null) {
          exportNames.add(classMatch[2]);
          namedExportLines.push(`export var ${classMatch[2]} = module.exports.${classMatch[2]};`);
        }
      }

      const postamble = [
        '',
        'export default (typeof module.exports !== "undefined" && module.exports !== exports) ? module.exports : exports;',
        ...namedExportLines,
        '',
      ].join('\n');

      return {
        code: preamble + transformed + postamble,
        map: null,
      };
    },
  };
}

export default defineConfig(({ mode }) => ({
  base: './',
  plugins: [
    cjsProtobufPlugin(),
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
  optimizeDeps: {
    include: [
      'google-protobuf',
      'google-protobuf/google/protobuf/any_pb',
      'google-protobuf/google/protobuf/struct_pb',
      'google-protobuf/google/protobuf/descriptor_pb',
      'google-protobuf/google/protobuf/field_mask_pb',
      'google-protobuf/google/protobuf/duration_pb',
      'grpc-web',
    ],
    esbuildOptions: {
      define: {
        global: 'globalThis',
      },
    },
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
