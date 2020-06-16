# -*- coding: utf-8 -*-
# Generated by the protocol buffer compiler.  DO NOT EDIT!
# source: go.chromium.org/luci/buildbucket/proto/builder.proto

from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from google.protobuf import reflection as _reflection
from google.protobuf import symbol_database as _symbol_database
# @@protoc_insertion_point(imports)

_sym_db = _symbol_database.Default()


from go.chromium.org.luci.buildbucket.proto import project_config_pb2 as go_dot_chromium_dot_org_dot_luci_dot_buildbucket_dot_proto_dot_project__config__pb2


DESCRIPTOR = _descriptor.FileDescriptor(
  name='go.chromium.org/luci/buildbucket/proto/builder.proto',
  package='buildbucket.v2',
  syntax='proto3',
  serialized_options=b'Z4go.chromium.org/luci/buildbucket/proto;buildbucketpb',
  create_key=_descriptor._internal_create_key,
  serialized_pb=b'\n4go.chromium.org/luci/buildbucket/proto/builder.proto\x12\x0e\x62uildbucket.v2\x1a;go.chromium.org/luci/buildbucket/proto/project_config.proto\"=\n\tBuilderID\x12\x0f\n\x07project\x18\x01 \x01(\t\x12\x0e\n\x06\x62ucket\x18\x02 \x01(\t\x12\x0f\n\x07\x62uilder\x18\x03 \x01(\t\"Z\n\x0b\x42uilderItem\x12%\n\x02id\x18\x01 \x01(\x0b\x32\x19.buildbucket.v2.BuilderID\x12$\n\x06\x63onfig\x18\x02 \x01(\x0b\x32\x14.buildbucket.BuilderB6Z4go.chromium.org/luci/buildbucket/proto;buildbucketpbb\x06proto3'
  ,
  dependencies=[go_dot_chromium_dot_org_dot_luci_dot_buildbucket_dot_proto_dot_project__config__pb2.DESCRIPTOR,])




_BUILDERID = _descriptor.Descriptor(
  name='BuilderID',
  full_name='buildbucket.v2.BuilderID',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  create_key=_descriptor._internal_create_key,
  fields=[
    _descriptor.FieldDescriptor(
      name='project', full_name='buildbucket.v2.BuilderID.project', index=0,
      number=1, type=9, cpp_type=9, label=1,
      has_default_value=False, default_value=b"".decode('utf-8'),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR,  create_key=_descriptor._internal_create_key),
    _descriptor.FieldDescriptor(
      name='bucket', full_name='buildbucket.v2.BuilderID.bucket', index=1,
      number=2, type=9, cpp_type=9, label=1,
      has_default_value=False, default_value=b"".decode('utf-8'),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR,  create_key=_descriptor._internal_create_key),
    _descriptor.FieldDescriptor(
      name='builder', full_name='buildbucket.v2.BuilderID.builder', index=2,
      number=3, type=9, cpp_type=9, label=1,
      has_default_value=False, default_value=b"".decode('utf-8'),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR,  create_key=_descriptor._internal_create_key),
  ],
  extensions=[
  ],
  nested_types=[],
  enum_types=[
  ],
  serialized_options=None,
  is_extendable=False,
  syntax='proto3',
  extension_ranges=[],
  oneofs=[
  ],
  serialized_start=133,
  serialized_end=194,
)


_BUILDERITEM = _descriptor.Descriptor(
  name='BuilderItem',
  full_name='buildbucket.v2.BuilderItem',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  create_key=_descriptor._internal_create_key,
  fields=[
    _descriptor.FieldDescriptor(
      name='id', full_name='buildbucket.v2.BuilderItem.id', index=0,
      number=1, type=11, cpp_type=10, label=1,
      has_default_value=False, default_value=None,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR,  create_key=_descriptor._internal_create_key),
    _descriptor.FieldDescriptor(
      name='config', full_name='buildbucket.v2.BuilderItem.config', index=1,
      number=2, type=11, cpp_type=10, label=1,
      has_default_value=False, default_value=None,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR,  create_key=_descriptor._internal_create_key),
  ],
  extensions=[
  ],
  nested_types=[],
  enum_types=[
  ],
  serialized_options=None,
  is_extendable=False,
  syntax='proto3',
  extension_ranges=[],
  oneofs=[
  ],
  serialized_start=196,
  serialized_end=286,
)

_BUILDERITEM.fields_by_name['id'].message_type = _BUILDERID
_BUILDERITEM.fields_by_name['config'].message_type = go_dot_chromium_dot_org_dot_luci_dot_buildbucket_dot_proto_dot_project__config__pb2._BUILDER
DESCRIPTOR.message_types_by_name['BuilderID'] = _BUILDERID
DESCRIPTOR.message_types_by_name['BuilderItem'] = _BUILDERITEM
_sym_db.RegisterFileDescriptor(DESCRIPTOR)

BuilderID = _reflection.GeneratedProtocolMessageType('BuilderID', (_message.Message,), {
  'DESCRIPTOR' : _BUILDERID,
  '__module__' : 'go.chromium.org.luci.buildbucket.proto.builder_pb2'
  # @@protoc_insertion_point(class_scope:buildbucket.v2.BuilderID)
  })
_sym_db.RegisterMessage(BuilderID)

BuilderItem = _reflection.GeneratedProtocolMessageType('BuilderItem', (_message.Message,), {
  'DESCRIPTOR' : _BUILDERITEM,
  '__module__' : 'go.chromium.org.luci.buildbucket.proto.builder_pb2'
  # @@protoc_insertion_point(class_scope:buildbucket.v2.BuilderItem)
  })
_sym_db.RegisterMessage(BuilderItem)


DESCRIPTOR._options = None
# @@protoc_insertion_point(module_scope)
