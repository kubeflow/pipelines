/**
 * Minimal protobuf codec for the MLMD lineage calls used by the screenshot
 * fixture seeder. Keeping this codec local lets the standalone utility run
 * without the frontend's generated google-protobuf dependencies.
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

function encodeContextIdRequest(contextId) {
  const value = BigInt(contextId);
  if (value <= 0n || value > MAX_INT64) {
    throw new Error(`MLMD context ID ${contextId} is outside the positive int64 range.`);
  }
  return Buffer.concat([Buffer.from([0x08]), encodeVarint(value, `MLMD context ID ${contextId}`)]);
}

function encodeExecutionIdsRequest(executionIds) {
  const fields = [];
  for (const executionId of executionIds) {
    const value = BigInt(executionId);
    if (value <= 0n || value > MAX_INT64) {
      throw new Error(`MLMD execution ID ${executionId} is outside the positive int64 range.`);
    }
    fields.push(Buffer.from([0x08]), encodeVarint(value, `MLMD execution ID ${executionId}`));
  }
  return Buffer.concat(fields);
}

function encodeStringField(fieldNumber, value, context) {
  if (typeof value !== 'string' || value.length === 0) {
    throw new Error(`${context} must be a nonempty string.`);
  }
  const bytes = Buffer.from(value, 'utf8');
  return Buffer.concat([
    encodeVarint(BigInt(fieldNumber << 3) | 2n, `${context} field tag`),
    encodeVarint(bytes.length, `${context} byte length`),
    bytes,
  ]);
}

function encodeGetContextByTypeAndNameRequest(typeName, contextName) {
  return Buffer.concat([
    encodeStringField(1, typeName, 'MLMD context type name'),
    encodeStringField(2, contextName, 'MLMD context name'),
  ]);
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

function positiveInt64String(value, context) {
  const signed = int64FromVarint(value, context);
  if (signed <= 0n) {
    throw new Error(`${context} must be a positive int64.`);
  }
  return signed.toString();
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

const EXECUTION_STATES = Object.freeze({
  0: 'UNKNOWN',
  1: 'NEW',
  2: 'RUNNING',
  3: 'COMPLETE',
  4: 'FAILED',
  5: 'CACHED',
  6: 'CANCELED',
});

const EVENT_TYPES = Object.freeze({
  0: 'UNKNOWN',
  1: 'DECLARED_OUTPUT',
  2: 'DECLARED_INPUT',
  3: 'INPUT',
  4: 'OUTPUT',
  5: 'INTERNAL_INPUT',
  6: 'INTERNAL_OUTPUT',
  7: 'PENDING_OUTPUT',
});

function decodeEnum(value, names, context) {
  const number = safeInt64Number(value, context);
  if (!Object.hasOwn(names, number)) {
    throw new Error(`${context} contains unsupported enum value ${number}.`);
  }
  return names[number];
}

function decodeExecution(bytes) {
  const reader = new ProtoReader(bytes, 'ml_metadata.Execution');
  let executionId = null;
  let name = null;
  let state = 'UNKNOWN';
  let type = null;
  const properties = {};
  const customProperties = {};
  while (!reader.done) {
    const { fieldNumber, wireType } = reader.readTag();
    if (fieldNumber === 1) {
      expectWireType(wireType, 0, 'ml_metadata.Execution.id');
      executionId = positiveInt64String(reader.readVarint('execution id'), 'MLMD execution ID');
    } else if (fieldNumber === 3) {
      expectWireType(wireType, 0, 'ml_metadata.Execution.last_known_state');
      state = decodeEnum(
        reader.readVarint('last_known_state'),
        EXECUTION_STATES,
        'ml_metadata.Execution.last_known_state',
      );
    } else if (fieldNumber === 4 || fieldNumber === 5) {
      expectWireType(wireType, 2, `ml_metadata.Execution field ${fieldNumber}`);
      const { key, value } = decodeArtifactProperty(reader.readBytes('property entry'));
      if (value !== undefined) {
        setOwn(fieldNumber === 4 ? properties : customProperties, key, value);
      }
    } else if (fieldNumber === 6) {
      expectWireType(wireType, 2, 'ml_metadata.Execution.name');
      name = reader.readString('execution name');
    } else if (fieldNumber === 7) {
      expectWireType(wireType, 2, 'ml_metadata.Execution.type');
      type = reader.readString('execution type');
    } else {
      reader.skipField(wireType);
    }
  }
  if (executionId === null) {
    throw new Error('ml_metadata.Execution is missing its required positive ID.');
  }
  return Object.fromEntries(
    Object.entries({
      executionId,
      metadata: { ...properties, ...customProperties },
      name,
      state,
      type,
    }).filter(([, value]) => value !== null),
  );
}

function decodeContext(bytes) {
  const reader = new ProtoReader(bytes, 'ml_metadata.Context');
  let contextId = null;
  let name = null;
  let type = null;
  while (!reader.done) {
    const { fieldNumber, wireType } = reader.readTag();
    if (fieldNumber === 1) {
      expectWireType(wireType, 0, 'ml_metadata.Context.id');
      contextId = positiveInt64String(reader.readVarint('context id'), 'MLMD context ID');
    } else if (fieldNumber === 3) {
      expectWireType(wireType, 2, 'ml_metadata.Context.name');
      name = reader.readString('context name');
    } else if (fieldNumber === 6) {
      expectWireType(wireType, 2, 'ml_metadata.Context.type');
      type = reader.readString('context type');
    } else {
      reader.skipField(wireType);
    }
  }
  if (contextId === null) {
    throw new Error('ml_metadata.Context is missing its required positive ID.');
  }
  return Object.fromEntries(
    Object.entries({ contextId, name, type }).filter(([, value]) => value !== null),
  );
}

function decodeEventPathStep(bytes) {
  const reader = new ProtoReader(bytes, 'ml_metadata.Event.Path.Step');
  let step = null;
  while (!reader.done) {
    const { fieldNumber, wireType } = reader.readTag();
    if (fieldNumber === 1) {
      expectWireType(wireType, 0, 'ml_metadata.Event.Path.Step.index');
      step = {
        index: int64FromVarint(reader.readVarint('path index'), 'MLMD event path index').toString(),
      };
    } else if (fieldNumber === 2) {
      expectWireType(wireType, 2, 'ml_metadata.Event.Path.Step.key');
      step = { key: reader.readString('path key') };
    } else {
      reader.skipField(wireType);
    }
  }
  if (step === null) {
    throw new Error('ml_metadata.Event.Path.Step does not contain an index or key.');
  }
  return step;
}

function decodeEventPath(bytes) {
  const reader = new ProtoReader(bytes, 'ml_metadata.Event.Path');
  const steps = [];
  while (!reader.done) {
    const { fieldNumber, wireType } = reader.readTag();
    if (fieldNumber === 1) {
      expectWireType(wireType, 2, 'ml_metadata.Event.Path.steps');
      steps.push(decodeEventPathStep(reader.readBytes('path step')));
    } else {
      reader.skipField(wireType);
    }
  }
  return steps;
}

function decodeEvent(bytes) {
  const reader = new ProtoReader(bytes, 'ml_metadata.Event');
  let artifactId = null;
  let executionId = null;
  let path = [];
  let type = 'UNKNOWN';
  while (!reader.done) {
    const { fieldNumber, wireType } = reader.readTag();
    if (fieldNumber === 1) {
      expectWireType(wireType, 0, 'ml_metadata.Event.artifact_id');
      artifactId = positiveInt64String(reader.readVarint('artifact id'), 'MLMD event artifact ID');
    } else if (fieldNumber === 2) {
      expectWireType(wireType, 0, 'ml_metadata.Event.execution_id');
      executionId = positiveInt64String(
        reader.readVarint('execution id'),
        'MLMD event execution ID',
      );
    } else if (fieldNumber === 3) {
      expectWireType(wireType, 2, 'ml_metadata.Event.path');
      path = decodeEventPath(reader.readBytes('event path'));
    } else if (fieldNumber === 4) {
      expectWireType(wireType, 0, 'ml_metadata.Event.type');
      type = decodeEnum(reader.readVarint('event type'), EVENT_TYPES, 'ml_metadata.Event.type');
    } else {
      reader.skipField(wireType);
    }
  }
  if (artifactId === null || executionId === null) {
    throw new Error('ml_metadata.Event is missing a required positive artifact or execution ID.');
  }
  return { artifactId, executionId, path, type };
}

function decodeArtifact(bytes) {
  const reader = new ProtoReader(bytes, 'ml_metadata.Artifact');
  let artifactId = null;
  let name = null;
  let type = null;
  let uri = null;
  const properties = {};
  const customProperties = {};
  while (!reader.done) {
    const { fieldNumber, wireType } = reader.readTag();
    if (fieldNumber === 1) {
      expectWireType(wireType, 0, 'ml_metadata.Artifact.id');
      artifactId = positiveInt64String(reader.readVarint('artifact id'), 'MLMD artifact ID');
    } else if (fieldNumber === 3) {
      expectWireType(wireType, 2, 'ml_metadata.Artifact.uri');
      uri = reader.readString('artifact uri');
    } else if (fieldNumber === 4 || fieldNumber === 5) {
      expectWireType(wireType, 2, `ml_metadata.Artifact field ${fieldNumber}`);
      const { key, value } = decodeArtifactProperty(reader.readBytes('property entry'));
      if (value !== undefined) {
        setOwn(fieldNumber === 4 ? properties : customProperties, key, value);
      }
    } else if (fieldNumber === 7) {
      expectWireType(wireType, 2, 'ml_metadata.Artifact.name');
      name = reader.readString('artifact name');
    } else if (fieldNumber === 8) {
      expectWireType(wireType, 2, 'ml_metadata.Artifact.type');
      type = reader.readString('artifact type');
    } else {
      reader.skipField(wireType);
    }
  }
  if (artifactId === null) {
    throw new Error('ml_metadata.Artifact is missing its required positive ID.');
  }
  return Object.fromEntries(
    Object.entries({
      artifactId,
      metadata: { ...properties, ...customProperties },
      name,
      type,
      uri,
    }).filter(([, value]) => value !== null),
  );
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

function decodeGetContextByTypeAndNameResponse(bytes) {
  const reader = new ProtoReader(bytes, 'ml_metadata.GetContextByTypeAndNameResponse');
  let context = null;
  while (!reader.done) {
    const { fieldNumber, wireType } = reader.readTag();
    if (fieldNumber === 1) {
      expectWireType(wireType, 2, 'ml_metadata.GetContextByTypeAndNameResponse.context');
      if (context !== null) {
        throw new Error(
          'ml_metadata.GetContextByTypeAndNameResponse contains more than one context.',
        );
      }
      context = decodeContext(reader.readBytes('context'));
    } else {
      reader.skipField(wireType);
    }
  }
  return context;
}

function decodeContextResponse(bytes, responseName, fieldName, decodeRecord) {
  const reader = new ProtoReader(bytes, `ml_metadata.${responseName}`);
  const records = [];
  let nextPageToken = '';
  while (!reader.done) {
    const { fieldNumber, wireType } = reader.readTag();
    if (fieldNumber === 1) {
      expectWireType(wireType, 2, `ml_metadata.${responseName}.${fieldName}`);
      records.push(decodeRecord(reader.readBytes(fieldName.slice(0, -1))));
    } else if (fieldNumber === 2) {
      expectWireType(wireType, 2, `ml_metadata.${responseName}.next_page_token`);
      nextPageToken = reader.readString('next_page_token');
    } else if (responseName === 'GetExecutionsByContextResponse' && fieldNumber === 3) {
      expectWireType(wireType, 2, 'ml_metadata.GetExecutionsByContextResponse.transaction_options');
      reader.readBytes('transaction_options');
    } else {
      reader.skipField(wireType);
    }
  }
  if (nextPageToken !== '') {
    throw new Error(
      `ml_metadata.${responseName} unexpectedly contains next_page_token; the unpaginated request would be incomplete.`,
    );
  }
  return records;
}

function decodeGetExecutionsByContextResponse(bytes) {
  return decodeContextResponse(
    bytes,
    'GetExecutionsByContextResponse',
    'executions',
    decodeExecution,
  );
}

function decodeGetArtifactsByContextResponse(bytes) {
  return decodeContextResponse(bytes, 'GetArtifactsByContextResponse', 'artifacts', decodeArtifact);
}

function decodeGetEventsByExecutionIdsResponse(bytes) {
  const reader = new ProtoReader(bytes, 'ml_metadata.GetEventsByExecutionIDsResponse');
  const events = [];
  while (!reader.done) {
    const { fieldNumber, wireType } = reader.readTag();
    if (fieldNumber === 1) {
      expectWireType(wireType, 2, 'ml_metadata.GetEventsByExecutionIDsResponse.events');
      events.push(decodeEvent(reader.readBytes('event')));
    } else {
      reader.skipField(wireType);
    }
  }
  return events;
}

module.exports = {
  decodeGetArtifactsByContextResponse,
  decodeGetArtifactsByIdResponse,
  decodeGetContextByTypeAndNameResponse,
  decodeGetEventsByExecutionIdsResponse,
  decodeGetExecutionsByContextResponse,
  encodeContextIdRequest,
  encodeExecutionIdsRequest,
  encodeGetContextByTypeAndNameRequest,
  encodeGetArtifactsByIdRequest,
};
