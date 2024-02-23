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


from google.protobuf import struct_pb2 as google_dot_protobuf_dot_struct__pb2


DESCRIPTOR = _descriptor_pool.Default().AddSerializedFile(
    b'\n\x13template_metadata.proto\x12\x11template_metadata\x1a\x1cgoogle/protobuf/struct.proto"F\n\x10TemplateMetadata\x12\x32\n\x0bio_metadata\x18\x01'
    b' \x01(\x0b\x32\x1d.template_metadata.IOMetadata"L\n\nIOMetadata\x12&\n\x05pages\x18\x01'
    b' \x03(\x0b\x32\x17.template_metadata.Page\x12\x16\n\x0eschema_version\x18\x02'
    b' \x01(\t"W\n\x04Page\x12\x0c\n\x04name\x18\x01'
    b' \x01(\t\x12\x13\n\x0b\x64\x65scription\x18\x02'
    b' \x01(\t\x12,\n\x08sections\x18\x03'
    b' \x03(\x0b\x32\x1a.template_metadata.Section"V\n\x07Section\x12\x0c\n\x04name\x18\x01'
    b' \x01(\t\x12\x13\n\x0b\x64\x65scription\x18\x02'
    b' \x01(\t\x12(\n\x06inputs\x18\x03'
    b' \x03(\x0b\x32\x18.template_metadata.Input"\x9a\x01\n\x05Input\x12\x14\n\x0c\x64isplay_name\x18\x01'
    b' \x01(\t\x12\x13\n\x0b\x64\x65scription\x18\x02'
    b' \x01(\t\x12\x1b\n\x13\x64\x65\x66\x61ult_explanation\x18\x03'
    b' \x01(\t\x12\x11\n\thelp_text\x18\x04'
    b' \x01(\t\x12\x36\n\rsemantic_type\x18\x05'
    b' \x01(\x0b\x32\x1f.template_metadata.SemanticType"\xf6\x02\n\x0cSemanticType\x12.\n\nfloat_type\x18\x01'
    b' \x01(\x0b\x32\x18.template_metadata.FloatH\x00\x12\x32\n\x0cinteger_type\x18\x02'
    b' \x01(\x0b\x32\x1a.template_metadata.IntegerH\x00\x12\x30\n\x0bstring_type\x18\x03'
    b' \x01(\x0b\x32\x19.template_metadata.StringH\x00\x12\x32\n\x0c\x62oolean_type\x18\x04'
    b' \x01(\x0b\x32\x1a.template_metadata.BooleanH\x00\x12,\n\tlist_type\x18\x06'
    b' \x01(\x0b\x32\x17.template_metadata.ListH\x00\x12\x30\n\x0bstruct_type\x18\x07'
    b' \x01(\x0b\x32\x19.template_metadata.StructH\x00\x12\x34\n\rartifact_type\x18\x08'
    b' \x01(\x0b\x32\x1b.template_metadata.ArtifactH\x00\x42\x06\n\x04type";\n\x05\x46loat\x12\x0b\n\x03min\x18\x01'
    b' \x01(\x02\x12\x0b\n\x03max\x18\x02'
    b' \x01(\x02\x12\x18\n\x10validation_error\x18\x03'
    b' \x01(\t"=\n\x07Integer\x12\x0b\n\x03min\x18\x01'
    b' \x01(\x05\x12\x0b\n\x03max\x18\x02'
    b' \x01(\x05\x12\x18\n\x10validation_error\x18\x03'
    b' \x01(\t"\xa6\x01\n\x06String\x12\x30\n\tfree_form\x18\x01'
    b' \x01(\x0b\x32\x1b.template_metadata.FreeFormH\x00\x12\x32\n\nselect_one\x18\x02'
    b' \x01(\x0b\x32\x1c.template_metadata.SelectOneH\x00\x12.\n\x08uri_type\x18\x03'
    b' \x01(\x0e\x32\x1a.template_metadata.UriTypeH\x00\x42\x06\n\x04type"\t\n\x07\x42oolean"\xa6\x01\n\x04List\x12\x30\n\tfree_form\x18\x01'
    b' \x01(\x0b\x32\x1b.template_metadata.FreeFormH\x00\x12\x34\n\x0bselect_many\x18\x02'
    b' \x01(\x0b\x32\x1d.template_metadata.SelectManyH\x00\x12.\n\x08uri_type\x18\x03'
    b' \x01(\x0e\x32\x1a.template_metadata.UriTypeH\x00\x42\x06\n\x04type"\x08\n\x06Struct"M\n\x08\x41rtifact\x12\'\n\x03uri\x18\x01'
    b' \x01(\x0e\x32\x1a.template_metadata.UriType\x12\x18\n\x10validation_error\x18\x02'
    b' \x01(\t"\x90\x01\n\x08\x46reeForm\x12%\n\x04size\x18\x01'
    b' \x01(\x0e\x32\x17.template_metadata.Size\x12\r\n\x05regex\x18\x02'
    b' \x01(\t\x12\x34\n\x0c\x63ontent_type\x18\x03'
    b' \x01(\x0e\x32\x1e.template_metadata.ContentType\x12\x18\n\x10validation_error\x18\x04'
    b' \x01(\t"\xbe\x01\n\tSelectOne\x12-\n\x07options\x18\x01'
    b' \x01(\x0b\x32\x1a.template_metadata.OptionsH\x00\x12/\n\x08location\x18\x02'
    b' \x01(\x0b\x32\x1b.template_metadata.LocationH\x00\x12\x11\n\x07project\x18\x03'
    b' \x01(\x08H\x00\x12\x36\n\x0cmachine_type\x18\x04'
    b' \x01(\x0b\x32\x1e.template_metadata.MachineTypeH\x00\x42\x06\n\x04type"K\n\nSelectMany\x12+\n\x07options\x18\x01'
    b' \x01(\x0b\x32\x1a.template_metadata.Options\x12\x10\n\x08select_n\x18\x02'
    b' \x01(\x05"R\n\x08Location\x12\r\n\x03\x61ny\x18\x01'
    b' \x01(\x08H\x00\x12-\n\x07options\x18\x02'
    b' \x01(\x0b\x32\x1a.template_metadata.OptionsH\x00\x42\x08\n\x06values"U\n\x0bMachineType\x12\r\n\x03\x61ny\x18\x01'
    b' \x01(\x08H\x00\x12-\n\x07options\x18\x02'
    b' \x01(\x0b\x32\x1a.template_metadata.OptionsH\x00\x42\x08\n\x06values"1\n\x07Options\x12&\n\x06values\x18\x01'
    b' \x03(\x0b\x32\x16.google.protobuf.Value*G\n\x04Size\x12\x0e\n\nSIZE_UNSET\x10\x00\x12\x0e\n\nSIZE_SMALL\x10\x01\x12\x0f\n\x0bSIZE_MEDIUM\x10\x02\x12\x0e\n\nSIZE_LARGE\x10\x03*\x82\x01\n\x0b\x43ontentType\x12\x11\n\rUNSET_CONTENT\x10\x00\x12\x10\n\x0cYAML_CONTENT\x10\x01\x12\x10\n\x0cJSON_CONTENT\x10\x02\x12\x14\n\x10MARKDOWN_CONTENT\x10\x03\x12\x10\n\x0cHTML_CONTENT\x10\x04\x12\x14\n\x10\x44\x41TETIME_CONTENT\x10\x05*a\n\x07UriType\x12\x0b\n\x07\x41NY_URI\x10\x00\x12\x0f\n\x0bGCS_ANY_URI\x10\x01\x12\x12\n\x0eGCS_BUCKET_URI\x10\x02\x12\x12\n\x0eGCS_OBJECT_URI\x10\x03\x12\x10\n\x0c\x42IGQUERY_URI\x10\x04\x42\x02P\x01\x62\x06proto3'
)

_globals = globals()
_builder.BuildMessageAndEnumDescriptors(DESCRIPTOR, _globals)
_builder.BuildTopDescriptorsAndMessages(
    DESCRIPTOR,
    'google_cloud_pipeline_components.google_cloud_pipeline_components.proto.template_metadata_pb2',
    _globals,
)
if not _descriptor._USE_C_DESCRIPTORS:
  _globals['DESCRIPTOR']._loaded_options = None
  _globals['DESCRIPTOR']._serialized_options = b'P\001'
  _globals['_SIZE']._serialized_start = 2225
  _globals['_SIZE']._serialized_end = 2296
  _globals['_CONTENTTYPE']._serialized_start = 2299
  _globals['_CONTENTTYPE']._serialized_end = 2429
  _globals['_URITYPE']._serialized_start = 2431
  _globals['_URITYPE']._serialized_end = 2528
  _globals['_TEMPLATEMETADATA']._serialized_start = 163
  _globals['_TEMPLATEMETADATA']._serialized_end = 233
  _globals['_IOMETADATA']._serialized_start = 235
  _globals['_IOMETADATA']._serialized_end = 311
  _globals['_PAGE']._serialized_start = 313
  _globals['_PAGE']._serialized_end = 400
  _globals['_SECTION']._serialized_start = 402
  _globals['_SECTION']._serialized_end = 488
  _globals['_INPUT']._serialized_start = 491
  _globals['_INPUT']._serialized_end = 645
  _globals['_SEMANTICTYPE']._serialized_start = 648
  _globals['_SEMANTICTYPE']._serialized_end = 1022
  _globals['_FLOAT']._serialized_start = 1024
  _globals['_FLOAT']._serialized_end = 1083
  _globals['_INTEGER']._serialized_start = 1085
  _globals['_INTEGER']._serialized_end = 1146
  _globals['_STRING']._serialized_start = 1149
  _globals['_STRING']._serialized_end = 1315
  _globals['_BOOLEAN']._serialized_start = 1317
  _globals['_BOOLEAN']._serialized_end = 1326
  _globals['_LIST']._serialized_start = 1329
  _globals['_LIST']._serialized_end = 1495
  _globals['_STRUCT']._serialized_start = 1497
  _globals['_STRUCT']._serialized_end = 1505
  _globals['_ARTIFACT']._serialized_start = 1507
  _globals['_ARTIFACT']._serialized_end = 1584
  _globals['_FREEFORM']._serialized_start = 1587
  _globals['_FREEFORM']._serialized_end = 1731
  _globals['_SELECTONE']._serialized_start = 1734
  _globals['_SELECTONE']._serialized_end = 1924
  _globals['_SELECTMANY']._serialized_start = 1926
  _globals['_SELECTMANY']._serialized_end = 2001
  _globals['_LOCATION']._serialized_start = 2003
  _globals['_LOCATION']._serialized_end = 2085
  _globals['_MACHINETYPE']._serialized_start = 2087
  _globals['_MACHINETYPE']._serialized_end = 2172
  _globals['_OPTIONS']._serialized_start = 2174
  _globals['_OPTIONS']._serialized_end = 2223
# @@protoc_insertion_point(module_scope)
