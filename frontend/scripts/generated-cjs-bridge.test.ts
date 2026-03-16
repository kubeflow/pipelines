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

import fs from 'node:fs';
import path from 'node:path';
import { describe, it, expect } from 'vitest';
import { transformGeneratedCjsToEsm, isGeneratedCjs } from '../vite-plugins/generated-cjs-bridge';

describe('isGeneratedCjs', () => {
  it('matches MLMD generated JS files', () => {
    expect(isGeneratedCjs('/abs/third_party/mlmd/generated/proto/foo_pb.js')).toBe(true);
  });

  it('matches src/generated JS files', () => {
    expect(isGeneratedCjs('/abs/src/generated/pipeline_spec/pipeline_spec_pb.js')).toBe(true);
  });

  it('rejects non-.js files in generated paths', () => {
    expect(isGeneratedCjs('/abs/src/generated/pipeline_spec/pipeline_spec.ts')).toBe(false);
  });

  it('rejects JS files outside generated paths', () => {
    expect(isGeneratedCjs('/abs/src/components/App.js')).toBe(false);
  });

  it('handles Vite query strings appended to module IDs', () => {
    expect(isGeneratedCjs('/abs/third_party/mlmd/generated/proto/foo_pb.js?t=123')).toBe(true);
  });

  it('handles Windows-style backslash paths', () => {
    expect(isGeneratedCjs('C:\\project\\src\\generated\\pipeline_spec\\pipeline_spec_pb.js')).toBe(
      true,
    );
  });
});

describe('transformGeneratedCjsToEsm', () => {
  it('converts var require to import (single quotes)', () => {
    const input = `var jspb = require('google-protobuf');`;
    const result = transformGeneratedCjsToEsm(input);
    expect(result.code).toContain(`import jspb from 'google-protobuf';`);
    expect(result.code).not.toContain('require');
    expect(result.requireMatches).toBe(1);
  });

  it('converts const require to import', () => {
    const input = `const proto = require('google-protobuf');`;
    const result = transformGeneratedCjsToEsm(input);
    expect(result.code).toContain(`import proto from 'google-protobuf';`);
    expect(result.requireMatches).toBe(1);
  });

  it('converts let require to import', () => {
    const input = `let jspb = require('google-protobuf');`;
    const result = transformGeneratedCjsToEsm(input);
    expect(result.code).toContain(`import jspb from 'google-protobuf';`);
    expect(result.requireMatches).toBe(1);
  });

  it('handles double-quoted require', () => {
    const input = `var jspb = require("google-protobuf");`;
    const result = transformGeneratedCjsToEsm(input);
    expect(result.code).toContain(`import jspb from 'google-protobuf';`);
    expect(result.requireMatches).toBe(1);
  });

  it('converts property-assignment require', () => {
    const input = `grpc.web = require('grpc-web');`;
    const result = transformGeneratedCjsToEsm(input);
    expect(result.code).toContain(`import __cjsImport0 from 'grpc-web';`);
    expect(result.code).toContain(`grpc.web = __cjsImport0;`);
    expect(result.requireMatches).toBe(1);
  });

  it('converts goog.object.extend(exports, ...) to ESM exports', () => {
    const input = [
      `goog.exportSymbol('proto.test.Foo', null, global);`,
      `goog.exportSymbol('proto.test.Bar', null, global);`,
      `goog.exportSymbol('proto.test.Foo.Nested', null, global);`,
      `goog.object.extend(exports, proto.test);`,
    ].join('\n');
    const result = transformGeneratedCjsToEsm(input);
    expect(result.code).toContain('export default proto.test;');
    expect(result.code).toContain('export const Foo = proto.test.Foo;');
    expect(result.code).toContain('export const Bar = proto.test.Bar;');
    expect(result.code).not.toMatch(/export const Nested\b/);
    expect(result.exportMatches).toBe(1);
    expect(result.exportSymbolNames).toContain('Foo');
    expect(result.exportSymbolNames).toContain('Bar');
  });

  it('converts module.exports to ESM exports', () => {
    const input = [
      `goog.exportSymbol('proto.ns.Alpha', null, global);`,
      `module.exports = proto.ns;`,
    ].join('\n');
    const result = transformGeneratedCjsToEsm(input);
    expect(result.code).toContain('export default proto.ns;');
    expect(result.code).toContain('export const Alpha = proto.ns.Alpha;');
    expect(result.exportMatches).toBe(1);
  });

  it('returns zero matches for a file with no CJS patterns', () => {
    const input = `export const x = 42;`;
    const result = transformGeneratedCjsToEsm(input);
    expect(result.requireMatches).toBe(0);
    expect(result.exportMatches).toBe(0);
  });

  it('handles optional semicolons', () => {
    const input = `var jspb = require('google-protobuf')`;
    const result = transformGeneratedCjsToEsm(input);
    expect(result.code).toContain(`import jspb from 'google-protobuf';`);
    expect(result.requireMatches).toBe(1);
  });

  it('handles double-quoted goog.exportSymbol', () => {
    const input = [
      `goog.exportSymbol("proto.ns.Thing", null, global);`,
      `goog.object.extend(exports, proto.ns);`,
    ].join('\n');
    const result = transformGeneratedCjsToEsm(input);
    expect(result.exportSymbolNames).toContain('Thing');
  });

  it('emits only one export default when both goog.object.extend and module.exports exist', () => {
    const input = [
      `goog.exportSymbol('proto.ns.Alpha', null, global);`,
      `goog.object.extend(exports, proto.ns);`,
      `module.exports = proto.ns;`,
    ].join('\n');
    const result = transformGeneratedCjsToEsm(input);
    const defaultCount = (result.code.match(/^export default /gm) || []).length;
    expect(defaultCount).toBe(1);
    expect(result.exportMatches).toBe(2);
  });

  it('falls back to namespace property assignments when no goog.exportSymbol', () => {
    const input = [
      `proto.svc.MyClient =`,
      `    function(hostname) { this.hostname = hostname; };`,
      `proto.svc.MyPromiseClient =`,
      `    function(hostname) { this.hostname = hostname; };`,
      `proto.svc.MyPromiseClient.prototype.doThing =`,
      `    function(request) { return null; };`,
      `module.exports = proto.svc;`,
    ].join('\n');
    const result = transformGeneratedCjsToEsm(input);
    expect(result.exportSymbolNames).toContain('MyClient');
    expect(result.exportSymbolNames).toContain('MyPromiseClient');
    expect(result.exportSymbolNames).not.toContain('prototype');
    expect(result.code).toContain('export const MyClient = proto.svc.MyClient;');
    expect(result.code).toContain('export const MyPromiseClient = proto.svc.MyPromiseClient;');
  });
});

describe('transformGeneratedCjsToEsm against real generated files', () => {
  const generatedFiles = [
    'src/third_party/mlmd/generated/ml_metadata/proto/metadata_store_pb.js',
    'src/third_party/mlmd/generated/ml_metadata/proto/metadata_store_service_pb.js',
    'src/third_party/mlmd/generated/ml_metadata/proto/metadata_store_service_grpc_web_pb.js',
    'src/generated/pipeline_spec/pipeline_spec_pb.js',
    'src/generated/pipeline_spec/google/rpc/status_pb.js',
    'src/generated/platform_spec/kubernetes_platform/kubernetes_executor_config_pb.js',
  ];

  const frontendRoot = path.resolve(__dirname, '..');

  for (const relPath of generatedFiles) {
    const fullPath = path.join(frontendRoot, relPath);

    it(`transforms ${relPath} with no remaining require() calls`, () => {
      const code = fs.readFileSync(fullPath, 'utf-8');
      const result = transformGeneratedCjsToEsm(code);

      expect(result.requireMatches).toBeGreaterThan(0);
      expect(result.exportMatches).toBeGreaterThan(0);

      const remainingRequires = result.code.match(/^(var|let|const)\s+\w+\s*=\s*require\(/gm);
      expect(remainingRequires).toBeNull();

      const remainingPropRequires = result.code.match(/^[\w.]+\s*=\s*require\(/gm);
      expect(remainingPropRequires).toBeNull();
    });

    it(`transforms ${relPath} with valid ESM export`, () => {
      const code = fs.readFileSync(fullPath, 'utf-8');
      const result = transformGeneratedCjsToEsm(code);

      expect(result.code).toMatch(/^export default /m);

      const noRemainingCjsExport =
        !result.code.match(/^goog\.object\.extend\(exports,/m) &&
        !result.code.match(/^module\.exports\s*=/m);
      expect(noRemainingCjsExport).toBe(true);
    });
  }
});
