// Copyright 2018-2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
import { readFileSync } from 'fs';
import { Transform, TransformOptions } from 'stream';

/** get the server address from host, port, and schema (defaults to 'http'). */
export function getAddress({
  host,
  port,
  namespace,
  schema = 'http',
}: {
  host: string;
  port?: string | number;
  namespace?: string;
  schema?: string;
}) {
  namespace = namespace ? `.${namespace}` : '';
  port = port ? `:${port}` : '';
  return `${schema}://${host}${namespace}${port}`;
}

export function equalArrays(a1: any[], a2: any[]): boolean {
  if (!Array.isArray(a1) || !Array.isArray(a2) || a1.length !== a2.length) {
    return false;
  }
  return JSON.stringify(a1) === JSON.stringify(a2);
}

export function generateRandomString(length: number): string {
  let d = new Date().getTime();
  function randomChar(): string {
    const r = Math.trunc((d + Math.random() * 16) % 16);
    d = Math.floor(d / 16);
    return r.toString(16);
  }
  let str = '';
  for (let i = 0; i < length; ++i) {
    str += randomChar();
  }
  return str;
}

export function loadJSON<T>(filepath?: string, defaultValue?: T): T | undefined {
  if (!filepath) {
    return defaultValue;
  }
  try {
    return JSON.parse(readFileSync(filepath, 'utf-8'));
  } catch (error) {
    console.error(`Failed reading json data from '${filepath}':`);
    console.error(error);
    return defaultValue;
  }
}

export interface PreviewStreamOptions extends TransformOptions {
  peek: number;
}

/**
 * Transform stream that only stream the first X number of bytes.
 */
export class PreviewStream extends Transform {
  _peek: number;

  constructor({ peek, ...opts }: PreviewStreamOptions) {
    // acts like passthrough
    let transform: TransformOptions['transform'] = (chunk, _encoding, callback) =>
      callback(undefined, chunk);
    // implements preview - peek must be positive number
    if (peek && peek > 0) {
      let size = 0;
      transform = (chunk, _encoding, callback) => {
        const delta = peek - size;
        size += chunk.length;
        if (size >= peek) {
          callback(undefined, chunk.slice(0, delta));
          this.resume(); // do not handle any subsequent data
          return;
        }
        callback(undefined, chunk);
      };
    }
    super({ ...opts, transform });
  }
}
