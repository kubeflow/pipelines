// Copyright 2026 The Kubeflow Authors
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

import { JSON_SCHEMA, load, Type } from 'js-yaml';

const YAML_1_1_INTEGER = /^[-+]?(?:0[bB][01]+|0[xX][0-9a-fA-F]+|0[oO][0-7]+|0[0-7]*|[1-9][0-9]*)$/;
const YAML_1_1_FLOAT = /^[-+]?(?:\.[0-9]+|[0-9]+(?:\.[0-9]*)?)(?:[eE][-+]?[0-9]+)?$/;
const YAML_BINARY = /^(?:[A-Za-z0-9+/]{4})*(?:[A-Za-z0-9+/]{2}==|[A-Za-z0-9+/]{3}=)?$/;
const YAML_TRUE_VALUES = new Set([
  'y',
  'Y',
  'yes',
  'Yes',
  'YES',
  'true',
  'True',
  'TRUE',
  'on',
  'On',
  'ON',
]);
const YAML_FALSE_VALUES = new Set([
  'n',
  'N',
  'no',
  'No',
  'NO',
  'false',
  'False',
  'FALSE',
  'off',
  'Off',
  'OFF',
]);
const YAML_POSITIVE_INFINITY_VALUES = new Set(['.inf', '.Inf', '.INF', '+.inf', '+.Inf', '+.INF']);
const YAML_NEGATIVE_INFINITY_VALUES = new Set(['-.inf', '-.Inf', '-.INF']);
const YAML_NAN_VALUES = new Set(['.nan', '.NaN', '.NAN']);
const YAML_UINT64_MAX = 18_446_744_073_709_551_615n;
const YAML_INT64_MIN_MAGNITUDE = 9_223_372_036_854_775_808n;

const LAUNCHER_YAML_SCHEMA = JSON_SCHEMA.extend({
  implicit: [
    new Type('tag:yaml.org,2002:merge', {
      kind: 'scalar',
      resolve: (value) => value === '<<' || value === null,
    }),
    new Type('tag:yaml.org,2002:bool', {
      kind: 'scalar',
      resolve: (value) => YAML_TRUE_VALUES.has(value) || YAML_FALSE_VALUES.has(value),
      construct: (value) => YAML_TRUE_VALUES.has(value),
    }),
    new Type('tag:yaml.org,2002:int', {
      kind: 'scalar',
      resolve: (value) => parseYamlInteger(value) !== undefined,
      construct: (value) => parseYamlInteger(value)!,
    }),
    new Type('tag:yaml.org,2002:float', {
      kind: 'scalar',
      resolve: (value) => {
        const normalized = value.replaceAll('_', '');
        return (
          YAML_POSITIVE_INFINITY_VALUES.has(normalized) ||
          YAML_NEGATIVE_INFINITY_VALUES.has(normalized) ||
          YAML_NAN_VALUES.has(normalized) ||
          (isYamlNumericCandidate(value) &&
            YAML_1_1_FLOAT.test(normalized) &&
            Number.isFinite(Number(normalized)))
        );
      },
      construct: (value) => {
        const normalized = value.replaceAll('_', '');
        if (YAML_POSITIVE_INFINITY_VALUES.has(normalized)) return Number.POSITIVE_INFINITY;
        if (YAML_NEGATIVE_INFINITY_VALUES.has(normalized)) return Number.NEGATIVE_INFINITY;
        if (YAML_NAN_VALUES.has(normalized)) return Number.NaN;
        return Number(normalized);
      },
    }),
  ],
  explicit: [
    new Type('tag:yaml.org,2002:binary', {
      kind: 'scalar',
      resolve: (value) => YAML_BINARY.test(value.replaceAll(/\s/g, '')),
      construct: (value) => Uint8Array.from(Buffer.from(value.replaceAll(/\s/g, ''), 'base64')),
    }),
  ],
});

export class LauncherConfigError extends Error {}

export class LauncherConfigParseError extends LauncherConfigError {
  constructor(message: string) {
    super(message);
    this.name = 'LauncherConfigParseError';
  }
}

export function parseGoYaml(value: string): unknown {
  return load(value, { schema: LAUNCHER_YAML_SCHEMA });
}

function parseYamlInteger(value: string): bigint | undefined {
  if (!isYamlNumericCandidate(value)) return undefined;
  const normalized = value.replaceAll('_', '');
  if (!YAML_1_1_INTEGER.test(normalized)) return undefined;
  const negative = normalized.startsWith('-');
  const unsigned = /^[+-]/.test(normalized) ? normalized.slice(1) : normalized;
  const radix = /^0[bB]/.test(unsigned)
    ? 2
    : /^0[xX]/.test(unsigned)
      ? 16
      : /^0[oO]/.test(unsigned) || /^0[0-7]+$/.test(unsigned)
        ? 8
        : 10;
  const digits = radix === 10 ? unsigned : unsigned.replace(/^0[bBoOxX]?/, '');
  const significantDigits = (digits || '0').replace(/^0+/, '') || '0';
  const maximumDigits = radix === 2 ? 64 : radix === 8 ? 22 : radix === 16 ? 16 : 20;
  if (significantDigits.length > maximumDigits) return undefined;
  let parsed = 0n;
  for (const digit of significantDigits) {
    parsed = parsed * BigInt(radix) + BigInt(Number.parseInt(digit, radix));
  }
  const limit = negative ? YAML_INT64_MIN_MAGNITUDE : YAML_UINT64_MAX;
  if (parsed > limit) return undefined;
  return negative ? -parsed : parsed;
}

function isYamlNumericCandidate(value: string): boolean {
  return /^[+\-0-9.]/.test(value);
}

export function formatGoStringScalar(value: number | bigint | boolean): string {
  if (value === Number.POSITIVE_INFINITY) return '+Inf';
  if (value === Number.NEGATIVE_INFINITY) return '-Inf';
  if (typeof value === 'number' && Number.isNaN(value)) return 'NaN';
  if (typeof value === 'number') return formatGoFloat32(value);
  return String(value);
}

// Mirrors strconv.FormatFloat(float64(f32), 'g', -1, 32) used by sigs.k8s.io/yaml.
export function formatGoFloat32(value: number): string {
  const float32 = Math.fround(value);
  if (Object.is(float32, -0)) return '-0';
  if (float32 === 0) return '0';
  if (Number.isNaN(float32)) return 'NaN';
  if (float32 === Number.POSITIVE_INFINITY) return '+Inf';
  if (float32 === Number.NEGATIVE_INFINITY) return '-Inf';

  const negative = float32 < 0;
  const magnitude = Math.abs(float32);
  let precision = 9;
  for (let candidatePrecision = 1; candidatePrecision <= 9; candidatePrecision++) {
    if (Math.fround(Number(magnitude.toPrecision(candidatePrecision))) === magnitude) {
      precision = candidatePrecision;
      break;
    }
  }

  const [rawCoefficient, rawExponent] = magnitude.toExponential(precision - 1).split('e');
  const decimalExponent = Number(rawExponent) - precision + 1;
  const roundedCoefficient = BigInt(rawCoefficient.replace('.', ''));
  const floatParts = getFloat32Parts(magnitude);
  let shortestCoefficient: bigint | undefined;
  let shortestDistance: bigint | undefined;
  for (const candidate of [roundedCoefficient - 1n, roundedCoefficient, roundedCoefficient + 1n]) {
    if (candidate <= 0n || Math.fround(Number(`${candidate}e${decimalExponent}`)) !== magnitude) {
      continue;
    }
    const distance = getFloat32DecimalDistance(candidate, decimalExponent, floatParts);
    if (
      shortestDistance === undefined ||
      distance < shortestDistance ||
      (distance === shortestDistance && candidate % 2n === 0n)
    ) {
      shortestCoefficient = candidate;
      shortestDistance = distance;
    }
  }

  let coefficient = shortestCoefficient!;
  let exponentAdjustment = decimalExponent;
  while (coefficient % 10n === 0n) {
    coefficient /= 10n;
    exponentAdjustment += 1;
  }

  const digits = String(coefficient);
  const exponent = digits.length - 1 + exponentAdjustment;
  const sign = negative ? '-' : '';
  if (exponent < -4 || exponent >= 6) {
    const scientificCoefficient = digits.length === 1 ? digits : `${digits[0]}.${digits.slice(1)}`;
    const exponentSign = exponent >= 0 ? '+' : '-';
    return `${sign}${scientificCoefficient}e${exponentSign}${String(Math.abs(exponent)).padStart(2, '0')}`;
  }

  const decimalPosition = digits.length + exponentAdjustment;
  if (decimalPosition <= 0) return `${sign}0.${'0'.repeat(-decimalPosition)}${digits}`;
  if (decimalPosition >= digits.length)
    return sign + digits + '0'.repeat(decimalPosition - digits.length);
  return `${sign}${digits.slice(0, decimalPosition)}.${digits.slice(decimalPosition)}`;
}

interface Float32Parts {
  binaryExponent: number;
  significand: bigint;
}

function getFloat32Parts(value: number): Float32Parts {
  const bytes = new ArrayBuffer(4);
  const view = new DataView(bytes);
  view.setFloat32(0, value);
  const bits = view.getUint32(0);
  const exponentBits = (bits >>> 23) & 0xff;
  const fractionBits = bits & 0x7fffff;
  return exponentBits === 0
    ? { binaryExponent: -149, significand: BigInt(fractionBits) }
    : { binaryExponent: exponentBits - 127 - 23, significand: BigInt(fractionBits + 0x800000) };
}

function getFloat32DecimalDistance(
  coefficient: bigint,
  decimalExponent: number,
  floatParts: Float32Parts,
): bigint {
  const { binaryExponent, significand } = floatParts;
  if (decimalExponent >= 0) {
    const integerCandidate = coefficient * 10n ** BigInt(decimalExponent);
    return binaryExponent >= 0
      ? absoluteBigInt(integerCandidate - (significand << BigInt(binaryExponent)))
      : absoluteBigInt((integerCandidate << BigInt(-binaryExponent)) - significand);
  }
  const decimalScale = 10n ** BigInt(-decimalExponent);
  return binaryExponent >= 0
    ? absoluteBigInt(coefficient - (significand << BigInt(binaryExponent)) * decimalScale)
    : absoluteBigInt((coefficient << BigInt(-binaryExponent)) - significand * decimalScale);
}

function absoluteBigInt(value: bigint): bigint {
  return value < 0n ? -value : value;
}

export function normalizeRecognizedKeys(
  value: Record<string, unknown>,
  recognizedKeys: readonly string[],
  path: string,
): Record<string, unknown> {
  const canonicalByLowerCase = new Map(
    recognizedKeys.map((key) => [key.toLowerCase(), key] as const),
  );
  const normalized: Record<string, unknown> = Object.create(null);
  const sourceKeyByCanonical = new Map<string, string>();
  for (const [sourceKey, entry] of Object.entries(value)) {
    const canonicalKey = canonicalByLowerCase.get(sourceKey.toLowerCase());
    if (canonicalKey === undefined || entry === null) continue;
    const previousSourceKey = sourceKeyByCanonical.get(canonicalKey);
    if (previousSourceKey !== undefined) {
      throw new LauncherConfigParseError(
        `kfp-launcher ${path} contains case-colliding keys ${previousSourceKey} and ${sourceKey}. ` +
          'Keep only one spelling and retry.',
      );
    }
    sourceKeyByCanonical.set(canonicalKey, sourceKey);
    normalized[canonicalKey] = entry;
  }
  return normalized;
}
