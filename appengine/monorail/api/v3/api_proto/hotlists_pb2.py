# Generated by the protocol buffer compiler.  DO NOT EDIT!
# source: api/v3/api_proto/hotlists.proto

import sys
_b=sys.version_info[0]<3 and (lambda x:x) or (lambda x:x.encode('latin1'))
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from google.protobuf import reflection as _reflection
from google.protobuf import symbol_database as _symbol_database
# @@protoc_insertion_point(imports)

_sym_db = _symbol_database.Default()


from api.v3.api_proto import feature_objects_pb2 as api_dot_v3_dot_api__proto_dot_feature__objects__pb2
from google.protobuf import field_mask_pb2 as google_dot_protobuf_dot_field__mask__pb2
from google.protobuf import empty_pb2 as google_dot_protobuf_dot_empty__pb2
from google_proto.google.api import field_behavior_pb2 as google__proto_dot_google_dot_api_dot_field__behavior__pb2
from google_proto.google.api import resource_pb2 as google__proto_dot_google_dot_api_dot_resource__pb2
from google_proto.google.api import annotations_pb2 as google__proto_dot_google_dot_api_dot_annotations__pb2


DESCRIPTOR = _descriptor.FileDescriptor(
  name='api/v3/api_proto/hotlists.proto',
  package='monorail.v3',
  syntax='proto3',
  serialized_options=None,
  serialized_pb=_b('\n\x1f\x61pi/v3/api_proto/hotlists.proto\x12\x0bmonorail.v3\x1a&api/v3/api_proto/feature_objects.proto\x1a google/protobuf/field_mask.proto\x1a\x1bgoogle/protobuf/empty.proto\x1a,google_proto/google/api/field_behavior.proto\x1a&google_proto/google/api/resource.proto\x1a)google_proto/google/api/annotations.proto\"B\n\x14\x43reateHotlistRequest\x12*\n\x07hotlist\x18\x01 \x01(\x0b\x32\x14.monorail.v3.HotlistB\x03\xe0\x41\x02\"@\n\x11GetHotlistRequest\x12+\n\x04name\x18\x01 \x01(\tB\x1d\xe0\x41\x02\xfa\x41\x17\n\x15\x61pi.crbug.com/Hotlist\"\x92\x01\n\x14UpdateHotlistRequest\x12\x44\n\x07hotlist\x18\x01 \x01(\x0b\x32\x14.monorail.v3.HotlistB\x1d\xe0\x41\x02\xfa\x41\x17\n\x15\x61pi.crbug.com/Hotlist\x12\x34\n\x0bupdate_mask\x18\x02 \x01(\x0b\x32\x1a.google.protobuf.FieldMaskB\x03\xe0\x41\x02\"\x81\x01\n\x17ListHotlistItemsRequest\x12-\n\x06parent\x18\x01 \x01(\tB\x1d\xe0\x41\x02\xfa\x41\x17\n\x15\x61pi.crbug.com/Hotlist\x12\x11\n\tpage_size\x18\x02 \x01(\x05\x12\x10\n\x08order_by\x18\x03 \x01(\t\x12\x12\n\npage_token\x18\x04 \x01(\t\"\\\n\x18ListHotlistItemsResponse\x12\'\n\x05items\x18\x01 \x03(\x0b\x32\x18.monorail.v3.HotlistItem\x12\x17\n\x0fnext_page_token\x18\x02 \x01(\t\"\xa0\x01\n\x19RerankHotlistItemsRequest\x12+\n\x04name\x18\x01 \x01(\tB\x1d\xfa\x41\x17\n\x15\x61pi.crbug.com/Hotlist\xe0\x41\x02\x12\x38\n\rhotlist_items\x18\x02 \x03(\tB!\xfa\x41\x1b\n\x19\x61pi.crbug.com/HotlistItem\xe0\x41\x02\x12\x1c\n\x0ftarget_position\x18\x03 \x01(\rB\x03\xe0\x41\x02\"\x8d\x01\n\x16\x41\x64\x64HotlistItemsRequest\x12-\n\x06parent\x18\x01 \x01(\tB\x1d\xe0\x41\x02\xfa\x41\x17\n\x15\x61pi.crbug.com/Hotlist\x12+\n\x06issues\x18\x02 \x03(\tB\x1b\xe0\x41\x02\xfa\x41\x15\n\x13\x61pi.crbug.com/Issue\x12\x17\n\x0ftarget_position\x18\x03 \x01(\r\"w\n\x19RemoveHotlistItemsRequest\x12-\n\x06parent\x18\x01 \x01(\tB\x1d\xe0\x41\x02\xfa\x41\x17\n\x15\x61pi.crbug.com/Hotlist\x12+\n\x06issues\x18\x02 \x03(\tB\x1b\xe0\x41\x02\xfa\x41\x15\n\x13\x61pi.crbug.com/Issue\"w\n\x1bRemoveHotlistEditorsRequest\x12+\n\x04name\x18\x01 \x01(\tB\x1d\xe0\x41\x02\xfa\x41\x17\n\x15\x61pi.crbug.com/Hotlist\x12+\n\x07\x65\x64itors\x18\x02 \x03(\tB\x1a\xe0\x41\x02\xfa\x41\x14\n\x12\x61pi.crbug.com/User\"H\n\x1cGatherHotlistsForUserRequest\x12(\n\x04user\x18\x01 \x01(\tB\x1a\xe0\x41\x02\xfa\x41\x14\n\x12\x61pi.crbug.com/User\"G\n\x1dGatherHotlistsForUserResponse\x12&\n\x08hotlists\x18\x01 \x03(\x0b\x32\x14.monorail.v3.Hotlist2\xe6\x06\n\x08Hotlists\x12J\n\rCreateHotlist\x12!.monorail.v3.CreateHotlistRequest\x1a\x14.monorail.v3.Hotlist\"\x00\x12\x44\n\nGetHotlist\x12\x1e.monorail.v3.GetHotlistRequest\x1a\x14.monorail.v3.Hotlist\"\x00\x12J\n\rUpdateHotlist\x12!.monorail.v3.UpdateHotlistRequest\x1a\x14.monorail.v3.Hotlist\"\x00\x12I\n\rDeleteHotlist\x12\x1e.monorail.v3.GetHotlistRequest\x1a\x16.google.protobuf.Empty\"\x00\x12\x61\n\x10ListHotlistItems\x12$.monorail.v3.ListHotlistItemsRequest\x1a%.monorail.v3.ListHotlistItemsResponse\"\x00\x12V\n\x12RerankHotlistItems\x12&.monorail.v3.RerankHotlistItemsRequest\x1a\x16.google.protobuf.Empty\"\x00\x12P\n\x0f\x41\x64\x64HotlistItems\x12#.monorail.v3.AddHotlistItemsRequest\x1a\x16.google.protobuf.Empty\"\x00\x12V\n\x12RemoveHotlistItems\x12&.monorail.v3.RemoveHotlistItemsRequest\x1a\x16.google.protobuf.Empty\"\x00\x12Z\n\x14RemoveHotlistEditors\x12(.monorail.v3.RemoveHotlistEditorsRequest\x1a\x16.google.protobuf.Empty\"\x00\x12p\n\x15GatherHotlistsForUser\x12).monorail.v3.GatherHotlistsForUserRequest\x1a*.monorail.v3.GatherHotlistsForUserResponse\"\x00\x62\x06proto3')
  ,
  dependencies=[api_dot_v3_dot_api__proto_dot_feature__objects__pb2.DESCRIPTOR,google_dot_protobuf_dot_field__mask__pb2.DESCRIPTOR,google_dot_protobuf_dot_empty__pb2.DESCRIPTOR,google__proto_dot_google_dot_api_dot_field__behavior__pb2.DESCRIPTOR,google__proto_dot_google_dot_api_dot_resource__pb2.DESCRIPTOR,google__proto_dot_google_dot_api_dot_annotations__pb2.DESCRIPTOR,])




_CREATEHOTLISTREQUEST = _descriptor.Descriptor(
  name='CreateHotlistRequest',
  full_name='monorail.v3.CreateHotlistRequest',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='hotlist', full_name='monorail.v3.CreateHotlistRequest.hotlist', index=0,
      number=1, type=11, cpp_type=10, label=1,
      has_default_value=False, default_value=None,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=_b('\340A\002'), file=DESCRIPTOR),
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
  serialized_start=280,
  serialized_end=346,
)


_GETHOTLISTREQUEST = _descriptor.Descriptor(
  name='GetHotlistRequest',
  full_name='monorail.v3.GetHotlistRequest',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='name', full_name='monorail.v3.GetHotlistRequest.name', index=0,
      number=1, type=9, cpp_type=9, label=1,
      has_default_value=False, default_value=_b("").decode('utf-8'),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=_b('\340A\002\372A\027\n\025api.crbug.com/Hotlist'), file=DESCRIPTOR),
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
  serialized_start=348,
  serialized_end=412,
)


_UPDATEHOTLISTREQUEST = _descriptor.Descriptor(
  name='UpdateHotlistRequest',
  full_name='monorail.v3.UpdateHotlistRequest',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='hotlist', full_name='monorail.v3.UpdateHotlistRequest.hotlist', index=0,
      number=1, type=11, cpp_type=10, label=1,
      has_default_value=False, default_value=None,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=_b('\340A\002\372A\027\n\025api.crbug.com/Hotlist'), file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='update_mask', full_name='monorail.v3.UpdateHotlistRequest.update_mask', index=1,
      number=2, type=11, cpp_type=10, label=1,
      has_default_value=False, default_value=None,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=_b('\340A\002'), file=DESCRIPTOR),
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
  serialized_start=415,
  serialized_end=561,
)


_LISTHOTLISTITEMSREQUEST = _descriptor.Descriptor(
  name='ListHotlistItemsRequest',
  full_name='monorail.v3.ListHotlistItemsRequest',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='parent', full_name='monorail.v3.ListHotlistItemsRequest.parent', index=0,
      number=1, type=9, cpp_type=9, label=1,
      has_default_value=False, default_value=_b("").decode('utf-8'),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=_b('\340A\002\372A\027\n\025api.crbug.com/Hotlist'), file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='page_size', full_name='monorail.v3.ListHotlistItemsRequest.page_size', index=1,
      number=2, type=5, cpp_type=1, label=1,
      has_default_value=False, default_value=0,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='order_by', full_name='monorail.v3.ListHotlistItemsRequest.order_by', index=2,
      number=3, type=9, cpp_type=9, label=1,
      has_default_value=False, default_value=_b("").decode('utf-8'),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='page_token', full_name='monorail.v3.ListHotlistItemsRequest.page_token', index=3,
      number=4, type=9, cpp_type=9, label=1,
      has_default_value=False, default_value=_b("").decode('utf-8'),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
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
  serialized_start=564,
  serialized_end=693,
)


_LISTHOTLISTITEMSRESPONSE = _descriptor.Descriptor(
  name='ListHotlistItemsResponse',
  full_name='monorail.v3.ListHotlistItemsResponse',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='items', full_name='monorail.v3.ListHotlistItemsResponse.items', index=0,
      number=1, type=11, cpp_type=10, label=3,
      has_default_value=False, default_value=[],
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='next_page_token', full_name='monorail.v3.ListHotlistItemsResponse.next_page_token', index=1,
      number=2, type=9, cpp_type=9, label=1,
      has_default_value=False, default_value=_b("").decode('utf-8'),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
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
  serialized_start=695,
  serialized_end=787,
)


_RERANKHOTLISTITEMSREQUEST = _descriptor.Descriptor(
  name='RerankHotlistItemsRequest',
  full_name='monorail.v3.RerankHotlistItemsRequest',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='name', full_name='monorail.v3.RerankHotlistItemsRequest.name', index=0,
      number=1, type=9, cpp_type=9, label=1,
      has_default_value=False, default_value=_b("").decode('utf-8'),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=_b('\372A\027\n\025api.crbug.com/Hotlist\340A\002'), file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='hotlist_items', full_name='monorail.v3.RerankHotlistItemsRequest.hotlist_items', index=1,
      number=2, type=9, cpp_type=9, label=3,
      has_default_value=False, default_value=[],
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=_b('\372A\033\n\031api.crbug.com/HotlistItem\340A\002'), file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='target_position', full_name='monorail.v3.RerankHotlistItemsRequest.target_position', index=2,
      number=3, type=13, cpp_type=3, label=1,
      has_default_value=False, default_value=0,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=_b('\340A\002'), file=DESCRIPTOR),
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
  serialized_start=790,
  serialized_end=950,
)


_ADDHOTLISTITEMSREQUEST = _descriptor.Descriptor(
  name='AddHotlistItemsRequest',
  full_name='monorail.v3.AddHotlistItemsRequest',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='parent', full_name='monorail.v3.AddHotlistItemsRequest.parent', index=0,
      number=1, type=9, cpp_type=9, label=1,
      has_default_value=False, default_value=_b("").decode('utf-8'),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=_b('\340A\002\372A\027\n\025api.crbug.com/Hotlist'), file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='issues', full_name='monorail.v3.AddHotlistItemsRequest.issues', index=1,
      number=2, type=9, cpp_type=9, label=3,
      has_default_value=False, default_value=[],
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=_b('\340A\002\372A\025\n\023api.crbug.com/Issue'), file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='target_position', full_name='monorail.v3.AddHotlistItemsRequest.target_position', index=2,
      number=3, type=13, cpp_type=3, label=1,
      has_default_value=False, default_value=0,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
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
  serialized_start=953,
  serialized_end=1094,
)


_REMOVEHOTLISTITEMSREQUEST = _descriptor.Descriptor(
  name='RemoveHotlistItemsRequest',
  full_name='monorail.v3.RemoveHotlistItemsRequest',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='parent', full_name='monorail.v3.RemoveHotlistItemsRequest.parent', index=0,
      number=1, type=9, cpp_type=9, label=1,
      has_default_value=False, default_value=_b("").decode('utf-8'),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=_b('\340A\002\372A\027\n\025api.crbug.com/Hotlist'), file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='issues', full_name='monorail.v3.RemoveHotlistItemsRequest.issues', index=1,
      number=2, type=9, cpp_type=9, label=3,
      has_default_value=False, default_value=[],
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=_b('\340A\002\372A\025\n\023api.crbug.com/Issue'), file=DESCRIPTOR),
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
  serialized_start=1096,
  serialized_end=1215,
)


_REMOVEHOTLISTEDITORSREQUEST = _descriptor.Descriptor(
  name='RemoveHotlistEditorsRequest',
  full_name='monorail.v3.RemoveHotlistEditorsRequest',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='name', full_name='monorail.v3.RemoveHotlistEditorsRequest.name', index=0,
      number=1, type=9, cpp_type=9, label=1,
      has_default_value=False, default_value=_b("").decode('utf-8'),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=_b('\340A\002\372A\027\n\025api.crbug.com/Hotlist'), file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='editors', full_name='monorail.v3.RemoveHotlistEditorsRequest.editors', index=1,
      number=2, type=9, cpp_type=9, label=3,
      has_default_value=False, default_value=[],
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=_b('\340A\002\372A\024\n\022api.crbug.com/User'), file=DESCRIPTOR),
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
  serialized_start=1217,
  serialized_end=1336,
)


_GATHERHOTLISTSFORUSERREQUEST = _descriptor.Descriptor(
  name='GatherHotlistsForUserRequest',
  full_name='monorail.v3.GatherHotlistsForUserRequest',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='user', full_name='monorail.v3.GatherHotlistsForUserRequest.user', index=0,
      number=1, type=9, cpp_type=9, label=1,
      has_default_value=False, default_value=_b("").decode('utf-8'),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=_b('\340A\002\372A\024\n\022api.crbug.com/User'), file=DESCRIPTOR),
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
  serialized_start=1338,
  serialized_end=1410,
)


_GATHERHOTLISTSFORUSERRESPONSE = _descriptor.Descriptor(
  name='GatherHotlistsForUserResponse',
  full_name='monorail.v3.GatherHotlistsForUserResponse',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='hotlists', full_name='monorail.v3.GatherHotlistsForUserResponse.hotlists', index=0,
      number=1, type=11, cpp_type=10, label=3,
      has_default_value=False, default_value=[],
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
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
  serialized_start=1412,
  serialized_end=1483,
)

_CREATEHOTLISTREQUEST.fields_by_name['hotlist'].message_type = api_dot_v3_dot_api__proto_dot_feature__objects__pb2._HOTLIST
_UPDATEHOTLISTREQUEST.fields_by_name['hotlist'].message_type = api_dot_v3_dot_api__proto_dot_feature__objects__pb2._HOTLIST
_UPDATEHOTLISTREQUEST.fields_by_name['update_mask'].message_type = google_dot_protobuf_dot_field__mask__pb2._FIELDMASK
_LISTHOTLISTITEMSRESPONSE.fields_by_name['items'].message_type = api_dot_v3_dot_api__proto_dot_feature__objects__pb2._HOTLISTITEM
_GATHERHOTLISTSFORUSERRESPONSE.fields_by_name['hotlists'].message_type = api_dot_v3_dot_api__proto_dot_feature__objects__pb2._HOTLIST
DESCRIPTOR.message_types_by_name['CreateHotlistRequest'] = _CREATEHOTLISTREQUEST
DESCRIPTOR.message_types_by_name['GetHotlistRequest'] = _GETHOTLISTREQUEST
DESCRIPTOR.message_types_by_name['UpdateHotlistRequest'] = _UPDATEHOTLISTREQUEST
DESCRIPTOR.message_types_by_name['ListHotlistItemsRequest'] = _LISTHOTLISTITEMSREQUEST
DESCRIPTOR.message_types_by_name['ListHotlistItemsResponse'] = _LISTHOTLISTITEMSRESPONSE
DESCRIPTOR.message_types_by_name['RerankHotlistItemsRequest'] = _RERANKHOTLISTITEMSREQUEST
DESCRIPTOR.message_types_by_name['AddHotlistItemsRequest'] = _ADDHOTLISTITEMSREQUEST
DESCRIPTOR.message_types_by_name['RemoveHotlistItemsRequest'] = _REMOVEHOTLISTITEMSREQUEST
DESCRIPTOR.message_types_by_name['RemoveHotlistEditorsRequest'] = _REMOVEHOTLISTEDITORSREQUEST
DESCRIPTOR.message_types_by_name['GatherHotlistsForUserRequest'] = _GATHERHOTLISTSFORUSERREQUEST
DESCRIPTOR.message_types_by_name['GatherHotlistsForUserResponse'] = _GATHERHOTLISTSFORUSERRESPONSE
_sym_db.RegisterFileDescriptor(DESCRIPTOR)

CreateHotlistRequest = _reflection.GeneratedProtocolMessageType('CreateHotlistRequest', (_message.Message,), dict(
  DESCRIPTOR = _CREATEHOTLISTREQUEST,
  __module__ = 'api.v3.api_proto.hotlists_pb2'
  # @@protoc_insertion_point(class_scope:monorail.v3.CreateHotlistRequest)
  ))
_sym_db.RegisterMessage(CreateHotlistRequest)

GetHotlistRequest = _reflection.GeneratedProtocolMessageType('GetHotlistRequest', (_message.Message,), dict(
  DESCRIPTOR = _GETHOTLISTREQUEST,
  __module__ = 'api.v3.api_proto.hotlists_pb2'
  # @@protoc_insertion_point(class_scope:monorail.v3.GetHotlistRequest)
  ))
_sym_db.RegisterMessage(GetHotlistRequest)

UpdateHotlistRequest = _reflection.GeneratedProtocolMessageType('UpdateHotlistRequest', (_message.Message,), dict(
  DESCRIPTOR = _UPDATEHOTLISTREQUEST,
  __module__ = 'api.v3.api_proto.hotlists_pb2'
  # @@protoc_insertion_point(class_scope:monorail.v3.UpdateHotlistRequest)
  ))
_sym_db.RegisterMessage(UpdateHotlistRequest)

ListHotlistItemsRequest = _reflection.GeneratedProtocolMessageType('ListHotlistItemsRequest', (_message.Message,), dict(
  DESCRIPTOR = _LISTHOTLISTITEMSREQUEST,
  __module__ = 'api.v3.api_proto.hotlists_pb2'
  # @@protoc_insertion_point(class_scope:monorail.v3.ListHotlistItemsRequest)
  ))
_sym_db.RegisterMessage(ListHotlistItemsRequest)

ListHotlistItemsResponse = _reflection.GeneratedProtocolMessageType('ListHotlistItemsResponse', (_message.Message,), dict(
  DESCRIPTOR = _LISTHOTLISTITEMSRESPONSE,
  __module__ = 'api.v3.api_proto.hotlists_pb2'
  # @@protoc_insertion_point(class_scope:monorail.v3.ListHotlistItemsResponse)
  ))
_sym_db.RegisterMessage(ListHotlistItemsResponse)

RerankHotlistItemsRequest = _reflection.GeneratedProtocolMessageType('RerankHotlistItemsRequest', (_message.Message,), dict(
  DESCRIPTOR = _RERANKHOTLISTITEMSREQUEST,
  __module__ = 'api.v3.api_proto.hotlists_pb2'
  # @@protoc_insertion_point(class_scope:monorail.v3.RerankHotlistItemsRequest)
  ))
_sym_db.RegisterMessage(RerankHotlistItemsRequest)

AddHotlistItemsRequest = _reflection.GeneratedProtocolMessageType('AddHotlistItemsRequest', (_message.Message,), dict(
  DESCRIPTOR = _ADDHOTLISTITEMSREQUEST,
  __module__ = 'api.v3.api_proto.hotlists_pb2'
  # @@protoc_insertion_point(class_scope:monorail.v3.AddHotlistItemsRequest)
  ))
_sym_db.RegisterMessage(AddHotlistItemsRequest)

RemoveHotlistItemsRequest = _reflection.GeneratedProtocolMessageType('RemoveHotlistItemsRequest', (_message.Message,), dict(
  DESCRIPTOR = _REMOVEHOTLISTITEMSREQUEST,
  __module__ = 'api.v3.api_proto.hotlists_pb2'
  # @@protoc_insertion_point(class_scope:monorail.v3.RemoveHotlistItemsRequest)
  ))
_sym_db.RegisterMessage(RemoveHotlistItemsRequest)

RemoveHotlistEditorsRequest = _reflection.GeneratedProtocolMessageType('RemoveHotlistEditorsRequest', (_message.Message,), dict(
  DESCRIPTOR = _REMOVEHOTLISTEDITORSREQUEST,
  __module__ = 'api.v3.api_proto.hotlists_pb2'
  # @@protoc_insertion_point(class_scope:monorail.v3.RemoveHotlistEditorsRequest)
  ))
_sym_db.RegisterMessage(RemoveHotlistEditorsRequest)

GatherHotlistsForUserRequest = _reflection.GeneratedProtocolMessageType('GatherHotlistsForUserRequest', (_message.Message,), dict(
  DESCRIPTOR = _GATHERHOTLISTSFORUSERREQUEST,
  __module__ = 'api.v3.api_proto.hotlists_pb2'
  # @@protoc_insertion_point(class_scope:monorail.v3.GatherHotlistsForUserRequest)
  ))
_sym_db.RegisterMessage(GatherHotlistsForUserRequest)

GatherHotlistsForUserResponse = _reflection.GeneratedProtocolMessageType('GatherHotlistsForUserResponse', (_message.Message,), dict(
  DESCRIPTOR = _GATHERHOTLISTSFORUSERRESPONSE,
  __module__ = 'api.v3.api_proto.hotlists_pb2'
  # @@protoc_insertion_point(class_scope:monorail.v3.GatherHotlistsForUserResponse)
  ))
_sym_db.RegisterMessage(GatherHotlistsForUserResponse)


_CREATEHOTLISTREQUEST.fields_by_name['hotlist']._options = None
_GETHOTLISTREQUEST.fields_by_name['name']._options = None
_UPDATEHOTLISTREQUEST.fields_by_name['hotlist']._options = None
_UPDATEHOTLISTREQUEST.fields_by_name['update_mask']._options = None
_LISTHOTLISTITEMSREQUEST.fields_by_name['parent']._options = None
_RERANKHOTLISTITEMSREQUEST.fields_by_name['name']._options = None
_RERANKHOTLISTITEMSREQUEST.fields_by_name['hotlist_items']._options = None
_RERANKHOTLISTITEMSREQUEST.fields_by_name['target_position']._options = None
_ADDHOTLISTITEMSREQUEST.fields_by_name['parent']._options = None
_ADDHOTLISTITEMSREQUEST.fields_by_name['issues']._options = None
_REMOVEHOTLISTITEMSREQUEST.fields_by_name['parent']._options = None
_REMOVEHOTLISTITEMSREQUEST.fields_by_name['issues']._options = None
_REMOVEHOTLISTEDITORSREQUEST.fields_by_name['name']._options = None
_REMOVEHOTLISTEDITORSREQUEST.fields_by_name['editors']._options = None
_GATHERHOTLISTSFORUSERREQUEST.fields_by_name['user']._options = None

_HOTLISTS = _descriptor.ServiceDescriptor(
  name='Hotlists',
  full_name='monorail.v3.Hotlists',
  file=DESCRIPTOR,
  index=0,
  serialized_options=None,
  serialized_start=1486,
  serialized_end=2356,
  methods=[
  _descriptor.MethodDescriptor(
    name='CreateHotlist',
    full_name='monorail.v3.Hotlists.CreateHotlist',
    index=0,
    containing_service=None,
    input_type=_CREATEHOTLISTREQUEST,
    output_type=api_dot_v3_dot_api__proto_dot_feature__objects__pb2._HOTLIST,
    serialized_options=None,
  ),
  _descriptor.MethodDescriptor(
    name='GetHotlist',
    full_name='monorail.v3.Hotlists.GetHotlist',
    index=1,
    containing_service=None,
    input_type=_GETHOTLISTREQUEST,
    output_type=api_dot_v3_dot_api__proto_dot_feature__objects__pb2._HOTLIST,
    serialized_options=None,
  ),
  _descriptor.MethodDescriptor(
    name='UpdateHotlist',
    full_name='monorail.v3.Hotlists.UpdateHotlist',
    index=2,
    containing_service=None,
    input_type=_UPDATEHOTLISTREQUEST,
    output_type=api_dot_v3_dot_api__proto_dot_feature__objects__pb2._HOTLIST,
    serialized_options=None,
  ),
  _descriptor.MethodDescriptor(
    name='DeleteHotlist',
    full_name='monorail.v3.Hotlists.DeleteHotlist',
    index=3,
    containing_service=None,
    input_type=_GETHOTLISTREQUEST,
    output_type=google_dot_protobuf_dot_empty__pb2._EMPTY,
    serialized_options=None,
  ),
  _descriptor.MethodDescriptor(
    name='ListHotlistItems',
    full_name='monorail.v3.Hotlists.ListHotlistItems',
    index=4,
    containing_service=None,
    input_type=_LISTHOTLISTITEMSREQUEST,
    output_type=_LISTHOTLISTITEMSRESPONSE,
    serialized_options=None,
  ),
  _descriptor.MethodDescriptor(
    name='RerankHotlistItems',
    full_name='monorail.v3.Hotlists.RerankHotlistItems',
    index=5,
    containing_service=None,
    input_type=_RERANKHOTLISTITEMSREQUEST,
    output_type=google_dot_protobuf_dot_empty__pb2._EMPTY,
    serialized_options=None,
  ),
  _descriptor.MethodDescriptor(
    name='AddHotlistItems',
    full_name='monorail.v3.Hotlists.AddHotlistItems',
    index=6,
    containing_service=None,
    input_type=_ADDHOTLISTITEMSREQUEST,
    output_type=google_dot_protobuf_dot_empty__pb2._EMPTY,
    serialized_options=None,
  ),
  _descriptor.MethodDescriptor(
    name='RemoveHotlistItems',
    full_name='monorail.v3.Hotlists.RemoveHotlistItems',
    index=7,
    containing_service=None,
    input_type=_REMOVEHOTLISTITEMSREQUEST,
    output_type=google_dot_protobuf_dot_empty__pb2._EMPTY,
    serialized_options=None,
  ),
  _descriptor.MethodDescriptor(
    name='RemoveHotlistEditors',
    full_name='monorail.v3.Hotlists.RemoveHotlistEditors',
    index=8,
    containing_service=None,
    input_type=_REMOVEHOTLISTEDITORSREQUEST,
    output_type=google_dot_protobuf_dot_empty__pb2._EMPTY,
    serialized_options=None,
  ),
  _descriptor.MethodDescriptor(
    name='GatherHotlistsForUser',
    full_name='monorail.v3.Hotlists.GatherHotlistsForUser',
    index=9,
    containing_service=None,
    input_type=_GATHERHOTLISTSFORUSERREQUEST,
    output_type=_GATHERHOTLISTSFORUSERRESPONSE,
    serialized_options=None,
  ),
])
_sym_db.RegisterServiceDescriptor(_HOTLISTS)

DESCRIPTOR.services_by_name['Hotlists'] = _HOTLISTS

# @@protoc_insertion_point(module_scope)
