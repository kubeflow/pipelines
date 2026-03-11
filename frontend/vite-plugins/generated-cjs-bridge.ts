/*
 * Copyright 2026 The Kubeflow Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import type { Plugin } from 'vite';

/**
 * Paths that contain generated CJS protobuf files. If new generated
 * directories are added, extend this list.
 */
const GENERATED_CJS_PATH_MARKERS = ['third_party/mlmd/generated', '/src/generated/'] as const;

export function isGeneratedCjs(id: string): boolean {
  const cleanId = id.split(/[?#]/)[0].replace(/\\/g, '/');
  if (!cleanId.endsWith('.js')) return false;
  return GENERATED_CJS_PATH_MARKERS.some((marker) => cleanId.includes(marker));
}

export interface TransformResult {
  code: string;
  requireMatches: number;
  exportMatches: number;
  exportSymbolNames: string[];
}

/**
 * Transforms a generated CJS protobuf file's source code into ESM-compatible
 * syntax for Vite's dev server. Returns the transformed code along with match
 * counts that callers can use to detect silent failures (zero matches on a file
 * that should have had them).
 *
 * Handles:
 *   - `var|let|const X = require('...')` / `require("...")`
 *   - `EXPR.PROP = require('...')` / `require("...")`
 *   - `goog.object.extend(exports, NAMESPACE)`
 *   - `module.exports = NAMESPACE`
 *   - `goog.exportSymbol('full.name', ...)` → derives named ESM exports
 */
export function transformGeneratedCjsToEsm(code: string): TransformResult {
  let transformed = code;
  const hoistedImports: string[] = [];
  let importCounter = 0;
  let requireMatches = 0;
  let exportMatches = 0;

  // Determine the export namespace (e.g. "proto.ml_metadata") so we can
  // derive named exports from goog.exportSymbol calls.
  let namespace = '';
  const nsExtend = code.match(/goog\.object\.extend\(exports,\s*([\w.]+)\)/);
  if (nsExtend) {
    namespace = nsExtend[1];
  } else {
    const nsModule = code.match(/module\.exports\s*=\s*([\w.]+)/);
    if (nsModule) namespace = nsModule[1];
  }

  // Collect top-level export names from goog.exportSymbol calls.
  // 'proto.ml_metadata.Artifact'       → 'Artifact' (direct child, exported)
  // 'proto.ml_metadata.Artifact.State' → nested, skipped
  const exportSymbolNames: string[] = [];
  if (namespace) {
    const nsPrefix = namespace + '.';
    const symbolRegex = /goog\.exportSymbol\(\s*['"]([\w.]+)['"]\s*,/g;
    let match;
    while ((match = symbolRegex.exec(code)) !== null) {
      const fullName = match[1];
      if (fullName.startsWith(nsPrefix)) {
        const relative = fullName.slice(nsPrefix.length);
        if (!relative.includes('.')) {
          exportSymbolNames.push(relative);
        }
      }
    }

    // Fallback: grpc-web files define names via direct property assignment
    // (e.g. proto.ml_metadata.MetadataStoreServicePromiseClient = ...)
    // instead of goog.exportSymbol. Scan for namespace.Name = patterns
    // where Name is NOT followed by another dot (excludes .prototype).
    if (exportSymbolNames.length === 0) {
      const escapedNs = namespace.replace(/\./g, '\\.');
      const propRegex = new RegExp('^' + escapedNs + '\\.(\\w+)\\s*=', 'gm');
      let propMatch;
      while ((propMatch = propRegex.exec(code)) !== null) {
        exportSymbolNames.push(propMatch[1]);
      }
    }
  }
  const uniqueExportNames = [...new Set(exportSymbolNames)];

  // var/let/const VARNAME = require('MODULE') or require("MODULE")
  transformed = transformed.replace(
    /^(var|let|const)\s+(\w+)\s*=\s*require\(\s*(['"])(.*?)\3\s*\)\s*;?\s*$/gm,
    (_match, _kw, varName, _q, modPath) => {
      requireMatches++;
      hoistedImports.push(`import ${varName} from '${modPath}';`);
      return '';
    },
  );

  // EXPR.PROP = require('MODULE') or require("MODULE")
  transformed = transformed.replace(
    /^([\w.]+)\s*=\s*require\(\s*(['"])(.*?)\2\s*\)\s*;?\s*$/gm,
    (_match, expression, _q, modPath) => {
      requireMatches++;
      const tempVar = `__cjsImport${importCounter++}`;
      hoistedImports.push(`import ${tempVar} from '${modPath}';`);
      return `${expression} = ${tempVar};`;
    },
  );

  let defaultExported = false;

  function buildExportBlock(expr: string): string {
    const lines: string[] = [];
    if (!defaultExported) {
      lines.push(`export default ${expr};`);
      defaultExported = true;
    }
    for (const name of uniqueExportNames) {
      lines.push(`export const ${name} = ${expr}.${name};`);
    }
    return lines.join('\n');
  }

  // goog.object.extend(exports, EXPR)
  transformed = transformed.replace(
    /^goog\.object\.extend\(exports,\s*([\w.]+)\)\s*;?\s*$/gm,
    (_match, expr) => {
      exportMatches++;
      return buildExportBlock(expr);
    },
  );

  // module.exports = EXPR
  transformed = transformed.replace(
    /^module\.exports\s*=\s*([\w.]+)\s*;?\s*$/gm,
    (_match, expr) => {
      exportMatches++;
      return buildExportBlock(expr);
    },
  );

  if (hoistedImports.length > 0) {
    transformed = hoistedImports.join('\n') + '\n' + transformed;
  }

  return {
    code: transformed,
    requireMatches,
    exportMatches,
    exportSymbolNames: uniqueExportNames,
  };
}

/**
 * Vite plugin that transforms generated protobuf JS files from CommonJS to ESM
 * during development. These files use Google Closure Library patterns
 * (goog.exportSymbol, goog.object.extend) that standard CJS-to-ESM plugins
 * cannot handle. Only active in dev (`serve`) mode; the production build uses
 * Rollup's @rollup/plugin-commonjs via `build.commonjsOptions` instead.
 */
export function generatedCjsBridge(): Plugin {
  return {
    name: 'generated-cjs-bridge',
    enforce: 'pre',
    apply: 'serve',

    transform(code, id) {
      if (!isGeneratedCjs(id)) {
        return null;
      }

      const result = transformGeneratedCjsToEsm(code);

      if (result.requireMatches === 0 && result.exportMatches === 0) {
        this.warn(
          `[generated-cjs-bridge] Processed ${id} but found no CJS patterns ` +
            `(0 require, 0 export). The generated file format may have changed. ` +
            `Verify the protobuf generators still produce the expected output.`,
        );
      }

      return { code: result.code, map: null };
    },
  };
}
