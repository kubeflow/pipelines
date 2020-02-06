// Copyright 2018 Google LLC
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
    return defaultValue;
  }
}
