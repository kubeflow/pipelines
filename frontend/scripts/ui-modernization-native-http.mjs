/*
 * Copyright 2026 The Kubeflow Authors
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at https://www.apache.org/licenses/LICENSE-2.0
 */

import assert from 'node:assert/strict';
import { request } from 'node:http';

function errorChain(error) {
  const parts = [];
  for (let current = error; current && parts.length < 5; current = current.cause) {
    parts.push(
      `${current.name || 'Error'}${current.code ? ` [${current.code}]` : ''}: ${current.message || current}`,
    );
  }
  return parts.join('; caused by ');
}

// Use the explicit command deadline for connection, headers AND body. Native
// fetch has a separate Undici header deadline that can end a slow WDA startup
// before the longer session AbortSignal expires.
export async function requestWebDriver(url, { method, body, timeout }) {
  url = new URL(url);
  assert.equal(url.protocol, 'http:');
  assert.ok(
    ['127.0.0.1', 'localhost', '[::1]'].includes(url.hostname),
    'WebDriver must be loopback',
  );
  assert.ok(Number.isInteger(timeout) && timeout > 0, 'A positive command deadline is required');
  const payload = body === undefined ? undefined : JSON.stringify(body);
  const signal = AbortSignal.timeout(timeout);
  const context = `${method} ${url.pathname}`;
  try {
    return await new Promise((resolve, reject) => {
      const req = request(
        url,
        {
          method,
          signal,
          agent: false,
          headers:
            payload === undefined
              ? undefined
              : {
                  'content-type': 'application/json',
                  'content-length': Buffer.byteLength(payload),
                },
        },
        (response) => {
          const chunks = [];
          let bytes = 0;
          response.on('data', (chunk) => {
            bytes += chunk.length;
            if (bytes > 64 * 1024 * 1024) {
              response.destroy(new Error('WebDriver response exceeded 64 MiB'));
            } else chunks.push(chunk);
          });
          response.once('error', reject);
          response.once('end', () => {
            try {
              const data = JSON.parse(Buffer.concat(chunks).toString('utf8'));
              if (response.statusCode < 200 || response.statusCode >= 300 || data.value?.error) {
                throw new Error(
                  `${data.value?.error || response.statusCode}: ${data.value?.message || ''}`,
                );
              }
              resolve(data.value);
            } catch (error) {
              reject(error);
            }
          });
        },
      );
      req.once('error', reject);
      req.end(payload);
    });
  } catch (error) {
    throw new Error(`${context}: ${errorChain(error)}`, { cause: error });
  }
}
