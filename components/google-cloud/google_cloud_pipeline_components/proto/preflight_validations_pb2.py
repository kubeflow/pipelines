# -*- coding: utf-8 -*-
# Generated by the protocol buffer compiler.  DO NOT EDIT!
# Protobuf Python Version: 0.20240110.0
"""Generated protocol buffer code."""
from google.protobuf import descriptor as _descriptor
from google.protobuf import descriptor_pool as _descriptor_pool
from google.protobuf import symbol_database as _symbol_database
from google.protobuf.internal import builder as _builder
# @@protoc_insertion_point(imports)

_sym_db = _symbol_database.Default()


DESCRIPTOR = _descriptor_pool.Default().AddSerializedFile(
    b'\n\x13preflight_validations.proto\x12\x15preflight_validations"\x8e\x02\n\x0fValidationItems\x12R\n\x0esa_validations\x18\x01'
    b' \x03(\x0b\x32:.preflight_validations.GoogleCloudServiceAccountValidation\x12S\n\x11quota_validations\x18\x02'
    b' \x03(\x0b\x32\x38.preflight_validations.GoogleCloudProjectQuotaValidation\x12R\n\x0f\x61pi_validations\x18\x03'
    b' \x03(\x0b\x32\x39.preflight_validations.GoogleCloudApiEnablementValidation"\x87\x01\n!GoogleCloudProjectQuotaValidation\x12\x14\n\x0cservice_name\x18\x01'
    b' \x01(\t\x12\x14\n\x0cmetrics_name\x18\x02'
    b' \x01(\t\x12\x15\n\x0bint64_value\x18\x03'
    b' \x01(\x03H\x00\x12\x16\n\x0c\x64ouble_value\x18\x04'
    b' \x01(\x01H\x00\x42\x07\n\x05value"\x84\x01\n#GoogleCloudServiceAccountValidation\x12\x1e\n\x16\x64\x65\x66\x61ult_principal_name\x18\x01'
    b' \x01(\t\x12\x14\n\x0c\x63\x61n_override\x18\x02'
    b' \x01(\x08\x12\x13\n\x0bpermissions\x18\x03'
    b' \x03(\t\x12\x12\n\nrole_names\x18\x04'
    b' \x03(\t";\n"GoogleCloudApiEnablementValidation\x12\x15\n\rservice_names\x18\x01'
    b' \x03(\tB\x02P\x01\x62\x06proto3'
)

_globals = globals()
_builder.BuildMessageAndEnumDescriptors(DESCRIPTOR, _globals)
_builder.BuildTopDescriptorsAndMessages(
    DESCRIPTOR,
    'google_cloud_pipeline_components.google_cloud_pipeline_components.proto.preflight_validations_pb2',
    _globals,
)
if not _descriptor._USE_C_DESCRIPTORS:
  _globals['DESCRIPTOR']._loaded_options = None
  _globals['DESCRIPTOR']._serialized_options = b'P\001'
  _globals['_VALIDATIONITEMS']._serialized_start = 142
  _globals['_VALIDATIONITEMS']._serialized_end = 412
  _globals['_GOOGLECLOUDPROJECTQUOTAVALIDATION']._serialized_start = 415
  _globals['_GOOGLECLOUDPROJECTQUOTAVALIDATION']._serialized_end = 550
  _globals['_GOOGLECLOUDSERVICEACCOUNTVALIDATION']._serialized_start = 553
  _globals['_GOOGLECLOUDSERVICEACCOUNTVALIDATION']._serialized_end = 685
  _globals['_GOOGLECLOUDAPIENABLEMENTVALIDATION']._serialized_start = 687
  _globals['_GOOGLECLOUDAPIENABLEMENTVALIDATION']._serialized_end = 746
# @@protoc_insertion_point(module_scope)
