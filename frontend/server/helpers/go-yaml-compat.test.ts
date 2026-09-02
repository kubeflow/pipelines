// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

import { describe, expect, it } from 'vitest';
import fixtures from './go-float32-fixtures.json';
import {
  formatGoFloat32,
  formatGoStringScalar,
  normalizeRecognizedKeys,
  parseGoYaml,
  LauncherConfigParseError,
} from './go-yaml-compat.js';

describe('Go YAML compatibility', () => {
  it('resolves yaml.v2 booleans without accepting mixed-case spellings', () => {
    expect(parseGoYaml('a: yes\nb: Off\nc: yEs\nd: true')).toEqual({
      a: true,
      b: false,
      c: 'yEs',
      d: true,
    });
  });

  it('resolves yaml.v2 integer forms and keeps date-like scalars as strings', () => {
    expect(parseGoYaml('binary: 0b101\noctal: 077\nhex: 0x10\ndate: 2026-01-01')).toEqual({
      binary: 5n,
      octal: 63n,
      hex: 16n,
      date: '2026-01-01',
    });
  });

  it('supports merge keys and yaml.v2 special floats', () => {
    const parsed = parseGoYaml(
      'base: &base\n  endpoint: store\nentry:\n  <<: *base\n  region: .inf\nnan: .nan',
    ) as Record<string, any>;
    expect(parsed.entry).toEqual({ endpoint: 'store', region: Infinity });
    expect(Number.isNaN(parsed.nan)).toBe(true);
  });

  it('normalizes recognized keys, drops unknown and null leaves, and rejects collisions', () => {
    expect(
      normalizeRecognizedKeys(
        { ENDPOINT: 'store', ignored: true, region: null },
        ['endpoint', 'region'],
        'providers.s3',
      ),
    ).toEqual({
      endpoint: 'store',
    });
    expect(() =>
      normalizeRecognizedKeys({ endpoint: 'a', ENDPOINT: 'b' }, ['endpoint'], 'providers.s3'),
    ).toThrow(LauncherConfigParseError);
  });

  it.each([
    [0, '0'],
    [-0, '-0'],
    [1, '1'],
    [1.5, '1.5'],
    [25000, '25000'],
    [1234567, '1.234567e+06'],
    [1e-5, '1e-05'],
    [Infinity, '+Inf'],
    [-Infinity, '-Inf'],
  ])('formats %s like strconv.FormatFloat(float32, g, -1, 32)', (value, expected) => {
    expect(formatGoFloat32(value)).toBe(expected);
  });

  it('stringifies Go scalar values for target string fields', () => {
    expect(formatGoStringScalar(true)).toBe('true');
    expect(formatGoStringScalar(25000)).toBe('25000');
  });

  it.each(fixtures)(
    'matches the checked-in Go float32 oracle for bits $bits',
    ({ bits, expected }) => {
      const bytes = new ArrayBuffer(4);
      new DataView(bytes).setUint32(0, Number.parseInt(bits, 16));
      expect(formatGoFloat32(new DataView(bytes).getFloat32(0))).toBe(expected);
    },
  );
});
