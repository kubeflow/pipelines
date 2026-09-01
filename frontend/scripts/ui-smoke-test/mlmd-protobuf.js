/**
 * Minimal protobuf codec for the MLMD GetArtifactsByID call used by the
 * screenshot fixture seeder. Keeping this codec local lets the standalone
 * utility run without the frontend's generated google-protobuf dependencies.
 */

const MAX_FIELD_NUMBER = 0x1fffffff;
const MAX_UINT64 = (1n << 64n) - 1n;
const MIN_INT64 = -(1n << 63n);
const MAX_INT64 = (1n << 63n) - 1n;

function encodeVarint(value, context) {
  let remaining = BigInt(value);
  if (remaining < 0n || remaining > MAX_UINT64) {
    throw new Error(`${context} is outside the protobuf uint64 range.`);
  }

  const bytes = [];
  do {
    let byte = Number(remaining & 0x7fn);
    remaining >>= 7n;
    if (remaining !== 0n) byte |= 0x80;
    bytes.push(byte);
  } while (remaining !== 0n);
  return Buffer.from(bytes);
}

function encodeGetArtifactsByIdRequest(artifactIds) {
  const fields = [];
  for (const artifactId of artifactIds) {
    const value = BigInt(artifactId);
    if (value <= 0n || value > MAX_INT64) {
      throw new Error(`MLMD artifact ID ${artifactId} is outside the positive int64 range.`);
    }
    fields.push(Buffer.from([0x08]), encodeVarint(value, `MLMD artifact ID ${artifactId}`));
  }
  return Buffer.concat(fields);
}

class ProtoReader {
  constructor(bytes, context) {
    this.buffer = Buffer.from(bytes);
    this.context = context;
    this.offset = 0;
  }

  get done() {
    return this.offset === this.buffer.length;
  }

  readVarint(description = 'varint') {
    let value = 0n;
    for (let index = 0; index < 10; index++) {
      if (this.offset >= this.buffer.length) {
        throw new Error(`${this.context} ended inside ${description}.`);
      }
      const byte = this.buffer[this.offset++];
      if (index === 9 && byte > 1) {
        throw new Error(`${this.context} contains an overflowing ${description}.`);
      }
      value |= BigInt(byte & 0x7f) << BigInt(index * 7);
      if ((byte & 0x80) === 0) return value;
    }
    throw new Error(`${this.context} contains an unterminated ${description}.`);
  }

  readTag() {
    const tag = this.readVarint('field tag');
    const fieldNumber = Number(tag >> 3n);
    const wireType = Number(tag & 0x07n);
    if (fieldNumber < 1 || fieldNumber > MAX_FIELD_NUMBER) {
      throw new Error(`${this.context} contains invalid protobuf field number ${fieldNumber}.`);
    }
    return { fieldNumber, wireType };
  }

  readBytes(description = 'length-delimited field') {
    const length = this.readVarint(`${description} length`);
    const remaining = BigInt(this.buffer.length - this.offset);
    if (length > remaining) {
      throw new Error(
        `${this.context} ${description} claims ${length} bytes, but only ${remaining} remain.`,
      );
    }
    const end = this.offset + Number(length);
    const bytes = this.buffer.subarray(this.offset, end);
    this.offset = end;
    return bytes;
  }

  readDouble(description) {
    this.requireBytes(8, description);
    const value = this.buffer.readDoubleLE(this.offset);
    this.offset += 8;
    return value;
  }

  readString(description) {
    return this.readBytes(description).toString('utf8');
  }

  requireBytes(length, description) {
    if (this.offset + length > this.buffer.length) {
      throw new Error(`${this.context} ended inside ${description}.`);
    }
  }

  skipField(wireType) {
    switch (wireType) {
      case 0:
        this.readVarint('unknown varint field');
        return;
      case 1:
        this.requireBytes(8, 'unknown fixed64 field');
        this.offset += 8;
        return;
      case 2:
        this.readBytes('unknown length-delimited field');
        return;
      case 5:
        this.requireBytes(4, 'unknown fixed32 field');
        this.offset += 4;
        return;
      default:
        throw new Error(`${this.context} uses unsupported protobuf wire type ${wireType}.`);
    }
  }
}

function expectWireType(actual, expected, context) {
  if (actual !== expected) {
    throw new Error(`${context} uses wire type ${actual}, expected ${expected}.`);
  }
}

function int64FromVarint(value, context) {
  const signed = value > MAX_INT64 ? value - (MAX_UINT64 + 1n) : value;
  if (signed < MIN_INT64 || signed > MAX_INT64) {
    throw new Error(`${context} is outside the protobuf int64 range.`);
  }
  return signed;
}

function safeInt64Number(value, context) {
  const signed = int64FromVarint(value, context);
  const number = Number(signed);
  if (!Number.isSafeInteger(number)) {
    throw new Error(`${context} ${signed} cannot be represented safely as a JavaScript number.`);
  }
  return number;
}

function setOwn(target, key, value) {
  Object.defineProperty(target, key, {
    configurable: true,
    enumerable: true,
    value,
    writable: true,
  });
}

function decodeGoogleValue(bytes) {
  const reader = new ProtoReader(bytes, 'google.protobuf.Value');
  let hasValue = false;
  let value;
  while (!reader.done) {
    const { fieldNumber, wireType } = reader.readTag();
    switch (fieldNumber) {
      case 1:
        expectWireType(wireType, 0, 'google.protobuf.Value.null_value');
        reader.readVarint('null_value');
        value = null;
        hasValue = true;
        break;
      case 2:
        expectWireType(wireType, 1, 'google.protobuf.Value.number_value');
        value = reader.readDouble('number_value');
        hasValue = true;
        break;
      case 3:
        expectWireType(wireType, 2, 'google.protobuf.Value.string_value');
        value = reader.readString('string_value');
        hasValue = true;
        break;
      case 4:
        expectWireType(wireType, 0, 'google.protobuf.Value.bool_value');
        value = reader.readVarint('bool_value') !== 0n;
        hasValue = true;
        break;
      case 5:
        expectWireType(wireType, 2, 'google.protobuf.Value.struct_value');
        value = decodeStruct(reader.readBytes('struct_value'));
        hasValue = true;
        break;
      case 6:
        expectWireType(wireType, 2, 'google.protobuf.Value.list_value');
        value = decodeListValue(reader.readBytes('list_value'));
        hasValue = true;
        break;
      default:
        reader.skipField(wireType);
    }
  }
  if (!hasValue) throw new Error('google.protobuf.Value does not contain a supported value kind.');
  return value;
}

function decodeStructEntry(bytes) {
  const reader = new ProtoReader(bytes, 'google.protobuf.Struct map entry');
  let key = '';
  let hasValue = false;
  let value;
  while (!reader.done) {
    const { fieldNumber, wireType } = reader.readTag();
    if (fieldNumber === 1) {
      expectWireType(wireType, 2, 'google.protobuf.Struct map key');
      key = reader.readString('map key');
    } else if (fieldNumber === 2) {
      expectWireType(wireType, 2, 'google.protobuf.Struct map value');
      value = decodeGoogleValue(reader.readBytes('map value'));
      hasValue = true;
    } else {
      reader.skipField(wireType);
    }
  }
  if (!hasValue)
    throw new Error(`google.protobuf.Struct field ${JSON.stringify(key)} has no value.`);
  return { key, value };
}

function decodeStruct(bytes) {
  const reader = new ProtoReader(bytes, 'google.protobuf.Struct');
  const result = {};
  while (!reader.done) {
    const { fieldNumber, wireType } = reader.readTag();
    if (fieldNumber === 1) {
      expectWireType(wireType, 2, 'google.protobuf.Struct.fields');
      const { key, value } = decodeStructEntry(reader.readBytes('fields entry'));
      setOwn(result, key, value);
    } else {
      reader.skipField(wireType);
    }
  }
  return result;
}

function decodeListValue(bytes) {
  const reader = new ProtoReader(bytes, 'google.protobuf.ListValue');
  const result = [];
  while (!reader.done) {
    const { fieldNumber, wireType } = reader.readTag();
    if (fieldNumber === 1) {
      expectWireType(wireType, 2, 'google.protobuf.ListValue.values');
      result.push(decodeGoogleValue(reader.readBytes('list value')));
    } else {
      reader.skipField(wireType);
    }
  }
  return result;
}

function unwrapMlmdStruct(value) {
  const keys = Object.keys(value);
  if (keys.length === 1 && (keys[0] === 'list' || keys[0] === 'struct')) {
    return value[keys[0]];
  }
  return value;
}

function decodeMlmdValue(bytes) {
  const reader = new ProtoReader(bytes, 'ml_metadata.Value');
  let value;
  while (!reader.done) {
    const { fieldNumber, wireType } = reader.readTag();
    switch (fieldNumber) {
      case 1:
        expectWireType(wireType, 0, 'ml_metadata.Value.int_value');
        value = safeInt64Number(reader.readVarint('int_value'), 'MLMD int_value');
        break;
      case 2:
        expectWireType(wireType, 1, 'ml_metadata.Value.double_value');
        value = reader.readDouble('double_value');
        break;
      case 3:
        expectWireType(wireType, 2, 'ml_metadata.Value.string_value');
        value = reader.readString('string_value');
        break;
      case 4:
        expectWireType(wireType, 2, 'ml_metadata.Value.struct_value');
        value = unwrapMlmdStruct(decodeStruct(reader.readBytes('struct_value')));
        break;
      case 5:
        expectWireType(wireType, 2, 'ml_metadata.Value.proto_value');
        reader.readBytes('proto_value');
        value = undefined;
        break;
      case 6:
        expectWireType(wireType, 0, 'ml_metadata.Value.bool_value');
        value = reader.readVarint('bool_value') !== 0n;
        break;
      default:
        reader.skipField(wireType);
    }
  }
  return value;
}

function decodeArtifactProperty(bytes) {
  const reader = new ProtoReader(bytes, 'ml_metadata.Artifact property');
  let key = '';
  let value;
  while (!reader.done) {
    const { fieldNumber, wireType } = reader.readTag();
    if (fieldNumber === 1) {
      expectWireType(wireType, 2, 'ml_metadata.Artifact property key');
      key = reader.readString('property key');
    } else if (fieldNumber === 2) {
      expectWireType(wireType, 2, 'ml_metadata.Artifact property value');
      value = decodeMlmdValue(reader.readBytes('property value'));
    } else {
      reader.skipField(wireType);
    }
  }
  return { key, value };
}

function decodeArtifact(bytes) {
  const reader = new ProtoReader(bytes, 'ml_metadata.Artifact');
  let artifactId = '0';
  const properties = {};
  const customProperties = {};
  while (!reader.done) {
    const { fieldNumber, wireType } = reader.readTag();
    if (fieldNumber === 1) {
      expectWireType(wireType, 0, 'ml_metadata.Artifact.id');
      artifactId = int64FromVarint(reader.readVarint('artifact id'), 'MLMD artifact ID').toString();
    } else if (fieldNumber === 4 || fieldNumber === 5) {
      expectWireType(wireType, 2, `ml_metadata.Artifact field ${fieldNumber}`);
      const { key, value } = decodeArtifactProperty(reader.readBytes('property entry'));
      if (value !== undefined) {
        setOwn(fieldNumber === 4 ? properties : customProperties, key, value);
      }
    } else {
      reader.skipField(wireType);
    }
  }
  return { artifactId, metadata: { ...properties, ...customProperties } };
}

function decodeGetArtifactsByIdResponse(bytes) {
  const reader = new ProtoReader(bytes, 'ml_metadata.GetArtifactsByIDResponse');
  const artifacts = [];
  while (!reader.done) {
    const { fieldNumber, wireType } = reader.readTag();
    if (fieldNumber === 1) {
      expectWireType(wireType, 2, 'ml_metadata.GetArtifactsByIDResponse.artifacts');
      artifacts.push(decodeArtifact(reader.readBytes('artifact')));
    } else {
      reader.skipField(wireType);
    }
  }
  return artifacts;
}

module.exports = {
  decodeGetArtifactsByIdResponse,
  encodeGetArtifactsByIdRequest,
};
