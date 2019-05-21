# Generated by the protocol buffer compiler.  DO NOT EDIT!
# source: api/api_proto/users.proto

import sys
_b=sys.version_info[0]<3 and (lambda x:x) or (lambda x:x.encode('latin1'))
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from google.protobuf import reflection as _reflection
from google.protobuf import symbol_database as _symbol_database
# @@protoc_insertion_point(imports)

_sym_db = _symbol_database.Default()


from api.api_proto import user_objects_pb2 as api_dot_api__proto_dot_user__objects__pb2
from api.api_proto import common_pb2 as api_dot_api__proto_dot_common__pb2


DESCRIPTOR = _descriptor.FileDescriptor(
  name='api/api_proto/users.proto',
  package='monorail',
  syntax='proto3',
  serialized_options=None,
  serialized_pb=_b('\n\x19\x61pi/api_proto/users.proto\x12\x08monorail\x1a api/api_proto/user_objects.proto\x1a\x1a\x61pi/api_proto/common.proto\"R\n\x1aListReferencedUsersRequest\x12\x0e\n\x06\x65mails\x18\x02 \x03(\t\x12$\n\tuser_refs\x18\x03 \x03(\x0b\x32\x11.monorail.UserRef\"<\n\x1bListReferencedUsersResponse\x12\x1d\n\x05users\x18\x01 \x03(\x0b\x32\x0e.monorail.User\"5\n\x0eGetUserRequest\x12#\n\x08user_ref\x18\x02 \x01(\x0b\x32\x11.monorail.UserRef\"<\n\x15GetMembershipsRequest\x12#\n\x08user_ref\x18\x02 \x01(\x0b\x32\x11.monorail.UserRef\"?\n\x16GetMembershipsResponse\x12%\n\ngroup_refs\x18\x01 \x03(\x0b\x32\x11.monorail.UserRef\"=\n\x16GetSavedQueriesRequest\x12#\n\x08user_ref\x18\x02 \x01(\x0b\x32\x11.monorail.UserRef\"F\n\x17GetSavedQueriesResponse\x12+\n\rsaved_queries\x18\x01 \x03(\x0b\x32\x14.monorail.SavedQuery\">\n\x17GetUserStarCountRequest\x12#\n\x08user_ref\x18\x02 \x01(\x0b\x32\x11.monorail.UserRef\".\n\x18GetUserStarCountResponse\x12\x12\n\nstar_count\x18\x01 \x01(\r\"G\n\x0fStarUserRequest\x12#\n\x08user_ref\x18\x02 \x01(\x0b\x32\x11.monorail.UserRef\x12\x0f\n\x07starred\x18\x03 \x01(\x08\"&\n\x10StarUserResponse\x12\x12\n\nstar_count\x18\x01 \x01(\r\"7\n\x1fSetExpandPermsPreferenceRequest\x12\x14\n\x0c\x65xpand_perms\x18\x02 \x01(\x08\"\"\n SetExpandPermsPreferenceResponse\":\n\x13GetUserPrefsRequest\x12#\n\x08user_ref\x18\x02 \x01(\x0b\x32\x11.monorail.UserRef\">\n\x14GetUserPrefsResponse\x12&\n\x05prefs\x18\x01 \x03(\x0b\x32\x17.monorail.UserPrefValue\"b\n\x13SetUserPrefsRequest\x12#\n\x08user_ref\x18\x02 \x01(\x0b\x32\x11.monorail.UserRef\x12&\n\x05prefs\x18\x03 \x03(\x0b\x32\x17.monorail.UserPrefValue\"\x16\n\x14SetUserPrefsResponse\"*\n\x19InviteLinkedParentRequest\x12\r\n\x05\x65mail\x18\x02 \x01(\t\"\x1c\n\x1aInviteLinkedParentResponse\")\n\x18\x41\x63\x63\x65ptLinkedChildRequest\x12\r\n\x05\x65mail\x18\x02 \x01(\t\"\x1b\n\x19\x41\x63\x63\x65ptLinkedChildResponse\"\\\n\x15UnlinkAccountsRequest\x12!\n\x06parent\x18\x02 \x01(\x0b\x32\x11.monorail.UserRef\x12 \n\x05\x63hild\x18\x03 \x01(\x0b\x32\x11.monorail.UserRef\"\x18\n\x16UnlinkAccountsResponse2\xa8\x08\n\x05Users\x12\x35\n\x07GetUser\x12\x18.monorail.GetUserRequest\x1a\x0e.monorail.User\"\x00\x12\x64\n\x13ListReferencedUsers\x12$.monorail.ListReferencedUsersRequest\x1a%.monorail.ListReferencedUsersResponse\"\x00\x12U\n\x0eGetMemberships\x12\x1f.monorail.GetMembershipsRequest\x1a .monorail.GetMembershipsResponse\"\x00\x12X\n\x0fGetSavedQueries\x12 .monorail.GetSavedQueriesRequest\x1a!.monorail.GetSavedQueriesResponse\"\x00\x12[\n\x10GetUserStarCount\x12!.monorail.GetUserStarCountRequest\x1a\".monorail.GetUserStarCountResponse\"\x00\x12\x43\n\x08StarUser\x12\x19.monorail.StarUserRequest\x1a\x1a.monorail.StarUserResponse\"\x00\x12O\n\x0cGetUserPrefs\x12\x1d.monorail.GetUserPrefsRequest\x1a\x1e.monorail.GetUserPrefsResponse\"\x00\x12O\n\x0cSetUserPrefs\x12\x1d.monorail.SetUserPrefsRequest\x1a\x1e.monorail.SetUserPrefsResponse\"\x00\x12s\n\x18SetExpandPermsPreference\x12).monorail.SetExpandPermsPreferenceRequest\x1a*.monorail.SetExpandPermsPreferenceResponse\"\x00\x12\x61\n\x12InviteLinkedParent\x12#.monorail.InviteLinkedParentRequest\x1a$.monorail.InviteLinkedParentResponse\"\x00\x12^\n\x11\x41\x63\x63\x65ptLinkedChild\x12\".monorail.AcceptLinkedChildRequest\x1a#.monorail.AcceptLinkedChildResponse\"\x00\x12U\n\x0eUnlinkAccounts\x12\x1f.monorail.UnlinkAccountsRequest\x1a .monorail.UnlinkAccountsResponse\"\x00\x62\x06proto3')
  ,
  dependencies=[api_dot_api__proto_dot_user__objects__pb2.DESCRIPTOR,api_dot_api__proto_dot_common__pb2.DESCRIPTOR,])




_LISTREFERENCEDUSERSREQUEST = _descriptor.Descriptor(
  name='ListReferencedUsersRequest',
  full_name='monorail.ListReferencedUsersRequest',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='emails', full_name='monorail.ListReferencedUsersRequest.emails', index=0,
      number=2, type=9, cpp_type=9, label=3,
      has_default_value=False, default_value=[],
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='user_refs', full_name='monorail.ListReferencedUsersRequest.user_refs', index=1,
      number=3, type=11, cpp_type=10, label=3,
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
  serialized_start=101,
  serialized_end=183,
)


_LISTREFERENCEDUSERSRESPONSE = _descriptor.Descriptor(
  name='ListReferencedUsersResponse',
  full_name='monorail.ListReferencedUsersResponse',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='users', full_name='monorail.ListReferencedUsersResponse.users', index=0,
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
  serialized_start=185,
  serialized_end=245,
)


_GETUSERREQUEST = _descriptor.Descriptor(
  name='GetUserRequest',
  full_name='monorail.GetUserRequest',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='user_ref', full_name='monorail.GetUserRequest.user_ref', index=0,
      number=2, type=11, cpp_type=10, label=1,
      has_default_value=False, default_value=None,
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
  serialized_start=247,
  serialized_end=300,
)


_GETMEMBERSHIPSREQUEST = _descriptor.Descriptor(
  name='GetMembershipsRequest',
  full_name='monorail.GetMembershipsRequest',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='user_ref', full_name='monorail.GetMembershipsRequest.user_ref', index=0,
      number=2, type=11, cpp_type=10, label=1,
      has_default_value=False, default_value=None,
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
  serialized_start=302,
  serialized_end=362,
)


_GETMEMBERSHIPSRESPONSE = _descriptor.Descriptor(
  name='GetMembershipsResponse',
  full_name='monorail.GetMembershipsResponse',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='group_refs', full_name='monorail.GetMembershipsResponse.group_refs', index=0,
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
  serialized_start=364,
  serialized_end=427,
)


_GETSAVEDQUERIESREQUEST = _descriptor.Descriptor(
  name='GetSavedQueriesRequest',
  full_name='monorail.GetSavedQueriesRequest',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='user_ref', full_name='monorail.GetSavedQueriesRequest.user_ref', index=0,
      number=2, type=11, cpp_type=10, label=1,
      has_default_value=False, default_value=None,
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
  serialized_start=429,
  serialized_end=490,
)


_GETSAVEDQUERIESRESPONSE = _descriptor.Descriptor(
  name='GetSavedQueriesResponse',
  full_name='monorail.GetSavedQueriesResponse',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='saved_queries', full_name='monorail.GetSavedQueriesResponse.saved_queries', index=0,
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
  serialized_start=492,
  serialized_end=562,
)


_GETUSERSTARCOUNTREQUEST = _descriptor.Descriptor(
  name='GetUserStarCountRequest',
  full_name='monorail.GetUserStarCountRequest',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='user_ref', full_name='monorail.GetUserStarCountRequest.user_ref', index=0,
      number=2, type=11, cpp_type=10, label=1,
      has_default_value=False, default_value=None,
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
  serialized_end=626,
)


_GETUSERSTARCOUNTRESPONSE = _descriptor.Descriptor(
  name='GetUserStarCountResponse',
  full_name='monorail.GetUserStarCountResponse',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='star_count', full_name='monorail.GetUserStarCountResponse.star_count', index=0,
      number=1, type=13, cpp_type=3, label=1,
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
  serialized_start=628,
  serialized_end=674,
)


_STARUSERREQUEST = _descriptor.Descriptor(
  name='StarUserRequest',
  full_name='monorail.StarUserRequest',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='user_ref', full_name='monorail.StarUserRequest.user_ref', index=0,
      number=2, type=11, cpp_type=10, label=1,
      has_default_value=False, default_value=None,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='starred', full_name='monorail.StarUserRequest.starred', index=1,
      number=3, type=8, cpp_type=7, label=1,
      has_default_value=False, default_value=False,
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
  serialized_start=676,
  serialized_end=747,
)


_STARUSERRESPONSE = _descriptor.Descriptor(
  name='StarUserResponse',
  full_name='monorail.StarUserResponse',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='star_count', full_name='monorail.StarUserResponse.star_count', index=0,
      number=1, type=13, cpp_type=3, label=1,
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
  serialized_start=749,
  serialized_end=787,
)


_SETEXPANDPERMSPREFERENCEREQUEST = _descriptor.Descriptor(
  name='SetExpandPermsPreferenceRequest',
  full_name='monorail.SetExpandPermsPreferenceRequest',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='expand_perms', full_name='monorail.SetExpandPermsPreferenceRequest.expand_perms', index=0,
      number=2, type=8, cpp_type=7, label=1,
      has_default_value=False, default_value=False,
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
  serialized_start=789,
  serialized_end=844,
)


_SETEXPANDPERMSPREFERENCERESPONSE = _descriptor.Descriptor(
  name='SetExpandPermsPreferenceResponse',
  full_name='monorail.SetExpandPermsPreferenceResponse',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
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
  serialized_start=846,
  serialized_end=880,
)


_GETUSERPREFSREQUEST = _descriptor.Descriptor(
  name='GetUserPrefsRequest',
  full_name='monorail.GetUserPrefsRequest',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='user_ref', full_name='monorail.GetUserPrefsRequest.user_ref', index=0,
      number=2, type=11, cpp_type=10, label=1,
      has_default_value=False, default_value=None,
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
  serialized_start=882,
  serialized_end=940,
)


_GETUSERPREFSRESPONSE = _descriptor.Descriptor(
  name='GetUserPrefsResponse',
  full_name='monorail.GetUserPrefsResponse',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='prefs', full_name='monorail.GetUserPrefsResponse.prefs', index=0,
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
  serialized_start=942,
  serialized_end=1004,
)


_SETUSERPREFSREQUEST = _descriptor.Descriptor(
  name='SetUserPrefsRequest',
  full_name='monorail.SetUserPrefsRequest',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='user_ref', full_name='monorail.SetUserPrefsRequest.user_ref', index=0,
      number=2, type=11, cpp_type=10, label=1,
      has_default_value=False, default_value=None,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='prefs', full_name='monorail.SetUserPrefsRequest.prefs', index=1,
      number=3, type=11, cpp_type=10, label=3,
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
  serialized_start=1006,
  serialized_end=1104,
)


_SETUSERPREFSRESPONSE = _descriptor.Descriptor(
  name='SetUserPrefsResponse',
  full_name='monorail.SetUserPrefsResponse',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
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
  serialized_start=1106,
  serialized_end=1128,
)


_INVITELINKEDPARENTREQUEST = _descriptor.Descriptor(
  name='InviteLinkedParentRequest',
  full_name='monorail.InviteLinkedParentRequest',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='email', full_name='monorail.InviteLinkedParentRequest.email', index=0,
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
  serialized_start=1130,
  serialized_end=1172,
)


_INVITELINKEDPARENTRESPONSE = _descriptor.Descriptor(
  name='InviteLinkedParentResponse',
  full_name='monorail.InviteLinkedParentResponse',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
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
  serialized_start=1174,
  serialized_end=1202,
)


_ACCEPTLINKEDCHILDREQUEST = _descriptor.Descriptor(
  name='AcceptLinkedChildRequest',
  full_name='monorail.AcceptLinkedChildRequest',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='email', full_name='monorail.AcceptLinkedChildRequest.email', index=0,
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
  serialized_start=1204,
  serialized_end=1245,
)


_ACCEPTLINKEDCHILDRESPONSE = _descriptor.Descriptor(
  name='AcceptLinkedChildResponse',
  full_name='monorail.AcceptLinkedChildResponse',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
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
  serialized_start=1247,
  serialized_end=1274,
)


_UNLINKACCOUNTSREQUEST = _descriptor.Descriptor(
  name='UnlinkAccountsRequest',
  full_name='monorail.UnlinkAccountsRequest',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='parent', full_name='monorail.UnlinkAccountsRequest.parent', index=0,
      number=2, type=11, cpp_type=10, label=1,
      has_default_value=False, default_value=None,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='child', full_name='monorail.UnlinkAccountsRequest.child', index=1,
      number=3, type=11, cpp_type=10, label=1,
      has_default_value=False, default_value=None,
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
  serialized_start=1276,
  serialized_end=1368,
)


_UNLINKACCOUNTSRESPONSE = _descriptor.Descriptor(
  name='UnlinkAccountsResponse',
  full_name='monorail.UnlinkAccountsResponse',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
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
  serialized_start=1370,
  serialized_end=1394,
)

_LISTREFERENCEDUSERSREQUEST.fields_by_name['user_refs'].message_type = api_dot_api__proto_dot_common__pb2._USERREF
_LISTREFERENCEDUSERSRESPONSE.fields_by_name['users'].message_type = api_dot_api__proto_dot_user__objects__pb2._USER
_GETUSERREQUEST.fields_by_name['user_ref'].message_type = api_dot_api__proto_dot_common__pb2._USERREF
_GETMEMBERSHIPSREQUEST.fields_by_name['user_ref'].message_type = api_dot_api__proto_dot_common__pb2._USERREF
_GETMEMBERSHIPSRESPONSE.fields_by_name['group_refs'].message_type = api_dot_api__proto_dot_common__pb2._USERREF
_GETSAVEDQUERIESREQUEST.fields_by_name['user_ref'].message_type = api_dot_api__proto_dot_common__pb2._USERREF
_GETSAVEDQUERIESRESPONSE.fields_by_name['saved_queries'].message_type = api_dot_api__proto_dot_common__pb2._SAVEDQUERY
_GETUSERSTARCOUNTREQUEST.fields_by_name['user_ref'].message_type = api_dot_api__proto_dot_common__pb2._USERREF
_STARUSERREQUEST.fields_by_name['user_ref'].message_type = api_dot_api__proto_dot_common__pb2._USERREF
_GETUSERPREFSREQUEST.fields_by_name['user_ref'].message_type = api_dot_api__proto_dot_common__pb2._USERREF
_GETUSERPREFSRESPONSE.fields_by_name['prefs'].message_type = api_dot_api__proto_dot_user__objects__pb2._USERPREFVALUE
_SETUSERPREFSREQUEST.fields_by_name['user_ref'].message_type = api_dot_api__proto_dot_common__pb2._USERREF
_SETUSERPREFSREQUEST.fields_by_name['prefs'].message_type = api_dot_api__proto_dot_user__objects__pb2._USERPREFVALUE
_UNLINKACCOUNTSREQUEST.fields_by_name['parent'].message_type = api_dot_api__proto_dot_common__pb2._USERREF
_UNLINKACCOUNTSREQUEST.fields_by_name['child'].message_type = api_dot_api__proto_dot_common__pb2._USERREF
DESCRIPTOR.message_types_by_name['ListReferencedUsersRequest'] = _LISTREFERENCEDUSERSREQUEST
DESCRIPTOR.message_types_by_name['ListReferencedUsersResponse'] = _LISTREFERENCEDUSERSRESPONSE
DESCRIPTOR.message_types_by_name['GetUserRequest'] = _GETUSERREQUEST
DESCRIPTOR.message_types_by_name['GetMembershipsRequest'] = _GETMEMBERSHIPSREQUEST
DESCRIPTOR.message_types_by_name['GetMembershipsResponse'] = _GETMEMBERSHIPSRESPONSE
DESCRIPTOR.message_types_by_name['GetSavedQueriesRequest'] = _GETSAVEDQUERIESREQUEST
DESCRIPTOR.message_types_by_name['GetSavedQueriesResponse'] = _GETSAVEDQUERIESRESPONSE
DESCRIPTOR.message_types_by_name['GetUserStarCountRequest'] = _GETUSERSTARCOUNTREQUEST
DESCRIPTOR.message_types_by_name['GetUserStarCountResponse'] = _GETUSERSTARCOUNTRESPONSE
DESCRIPTOR.message_types_by_name['StarUserRequest'] = _STARUSERREQUEST
DESCRIPTOR.message_types_by_name['StarUserResponse'] = _STARUSERRESPONSE
DESCRIPTOR.message_types_by_name['SetExpandPermsPreferenceRequest'] = _SETEXPANDPERMSPREFERENCEREQUEST
DESCRIPTOR.message_types_by_name['SetExpandPermsPreferenceResponse'] = _SETEXPANDPERMSPREFERENCERESPONSE
DESCRIPTOR.message_types_by_name['GetUserPrefsRequest'] = _GETUSERPREFSREQUEST
DESCRIPTOR.message_types_by_name['GetUserPrefsResponse'] = _GETUSERPREFSRESPONSE
DESCRIPTOR.message_types_by_name['SetUserPrefsRequest'] = _SETUSERPREFSREQUEST
DESCRIPTOR.message_types_by_name['SetUserPrefsResponse'] = _SETUSERPREFSRESPONSE
DESCRIPTOR.message_types_by_name['InviteLinkedParentRequest'] = _INVITELINKEDPARENTREQUEST
DESCRIPTOR.message_types_by_name['InviteLinkedParentResponse'] = _INVITELINKEDPARENTRESPONSE
DESCRIPTOR.message_types_by_name['AcceptLinkedChildRequest'] = _ACCEPTLINKEDCHILDREQUEST
DESCRIPTOR.message_types_by_name['AcceptLinkedChildResponse'] = _ACCEPTLINKEDCHILDRESPONSE
DESCRIPTOR.message_types_by_name['UnlinkAccountsRequest'] = _UNLINKACCOUNTSREQUEST
DESCRIPTOR.message_types_by_name['UnlinkAccountsResponse'] = _UNLINKACCOUNTSRESPONSE
_sym_db.RegisterFileDescriptor(DESCRIPTOR)

ListReferencedUsersRequest = _reflection.GeneratedProtocolMessageType('ListReferencedUsersRequest', (_message.Message,), dict(
  DESCRIPTOR = _LISTREFERENCEDUSERSREQUEST,
  __module__ = 'api.api_proto.users_pb2'
  # @@protoc_insertion_point(class_scope:monorail.ListReferencedUsersRequest)
  ))
_sym_db.RegisterMessage(ListReferencedUsersRequest)

ListReferencedUsersResponse = _reflection.GeneratedProtocolMessageType('ListReferencedUsersResponse', (_message.Message,), dict(
  DESCRIPTOR = _LISTREFERENCEDUSERSRESPONSE,
  __module__ = 'api.api_proto.users_pb2'
  # @@protoc_insertion_point(class_scope:monorail.ListReferencedUsersResponse)
  ))
_sym_db.RegisterMessage(ListReferencedUsersResponse)

GetUserRequest = _reflection.GeneratedProtocolMessageType('GetUserRequest', (_message.Message,), dict(
  DESCRIPTOR = _GETUSERREQUEST,
  __module__ = 'api.api_proto.users_pb2'
  # @@protoc_insertion_point(class_scope:monorail.GetUserRequest)
  ))
_sym_db.RegisterMessage(GetUserRequest)

GetMembershipsRequest = _reflection.GeneratedProtocolMessageType('GetMembershipsRequest', (_message.Message,), dict(
  DESCRIPTOR = _GETMEMBERSHIPSREQUEST,
  __module__ = 'api.api_proto.users_pb2'
  # @@protoc_insertion_point(class_scope:monorail.GetMembershipsRequest)
  ))
_sym_db.RegisterMessage(GetMembershipsRequest)

GetMembershipsResponse = _reflection.GeneratedProtocolMessageType('GetMembershipsResponse', (_message.Message,), dict(
  DESCRIPTOR = _GETMEMBERSHIPSRESPONSE,
  __module__ = 'api.api_proto.users_pb2'
  # @@protoc_insertion_point(class_scope:monorail.GetMembershipsResponse)
  ))
_sym_db.RegisterMessage(GetMembershipsResponse)

GetSavedQueriesRequest = _reflection.GeneratedProtocolMessageType('GetSavedQueriesRequest', (_message.Message,), dict(
  DESCRIPTOR = _GETSAVEDQUERIESREQUEST,
  __module__ = 'api.api_proto.users_pb2'
  # @@protoc_insertion_point(class_scope:monorail.GetSavedQueriesRequest)
  ))
_sym_db.RegisterMessage(GetSavedQueriesRequest)

GetSavedQueriesResponse = _reflection.GeneratedProtocolMessageType('GetSavedQueriesResponse', (_message.Message,), dict(
  DESCRIPTOR = _GETSAVEDQUERIESRESPONSE,
  __module__ = 'api.api_proto.users_pb2'
  # @@protoc_insertion_point(class_scope:monorail.GetSavedQueriesResponse)
  ))
_sym_db.RegisterMessage(GetSavedQueriesResponse)

GetUserStarCountRequest = _reflection.GeneratedProtocolMessageType('GetUserStarCountRequest', (_message.Message,), dict(
  DESCRIPTOR = _GETUSERSTARCOUNTREQUEST,
  __module__ = 'api.api_proto.users_pb2'
  # @@protoc_insertion_point(class_scope:monorail.GetUserStarCountRequest)
  ))
_sym_db.RegisterMessage(GetUserStarCountRequest)

GetUserStarCountResponse = _reflection.GeneratedProtocolMessageType('GetUserStarCountResponse', (_message.Message,), dict(
  DESCRIPTOR = _GETUSERSTARCOUNTRESPONSE,
  __module__ = 'api.api_proto.users_pb2'
  # @@protoc_insertion_point(class_scope:monorail.GetUserStarCountResponse)
  ))
_sym_db.RegisterMessage(GetUserStarCountResponse)

StarUserRequest = _reflection.GeneratedProtocolMessageType('StarUserRequest', (_message.Message,), dict(
  DESCRIPTOR = _STARUSERREQUEST,
  __module__ = 'api.api_proto.users_pb2'
  # @@protoc_insertion_point(class_scope:monorail.StarUserRequest)
  ))
_sym_db.RegisterMessage(StarUserRequest)

StarUserResponse = _reflection.GeneratedProtocolMessageType('StarUserResponse', (_message.Message,), dict(
  DESCRIPTOR = _STARUSERRESPONSE,
  __module__ = 'api.api_proto.users_pb2'
  # @@protoc_insertion_point(class_scope:monorail.StarUserResponse)
  ))
_sym_db.RegisterMessage(StarUserResponse)

SetExpandPermsPreferenceRequest = _reflection.GeneratedProtocolMessageType('SetExpandPermsPreferenceRequest', (_message.Message,), dict(
  DESCRIPTOR = _SETEXPANDPERMSPREFERENCEREQUEST,
  __module__ = 'api.api_proto.users_pb2'
  # @@protoc_insertion_point(class_scope:monorail.SetExpandPermsPreferenceRequest)
  ))
_sym_db.RegisterMessage(SetExpandPermsPreferenceRequest)

SetExpandPermsPreferenceResponse = _reflection.GeneratedProtocolMessageType('SetExpandPermsPreferenceResponse', (_message.Message,), dict(
  DESCRIPTOR = _SETEXPANDPERMSPREFERENCERESPONSE,
  __module__ = 'api.api_proto.users_pb2'
  # @@protoc_insertion_point(class_scope:monorail.SetExpandPermsPreferenceResponse)
  ))
_sym_db.RegisterMessage(SetExpandPermsPreferenceResponse)

GetUserPrefsRequest = _reflection.GeneratedProtocolMessageType('GetUserPrefsRequest', (_message.Message,), dict(
  DESCRIPTOR = _GETUSERPREFSREQUEST,
  __module__ = 'api.api_proto.users_pb2'
  # @@protoc_insertion_point(class_scope:monorail.GetUserPrefsRequest)
  ))
_sym_db.RegisterMessage(GetUserPrefsRequest)

GetUserPrefsResponse = _reflection.GeneratedProtocolMessageType('GetUserPrefsResponse', (_message.Message,), dict(
  DESCRIPTOR = _GETUSERPREFSRESPONSE,
  __module__ = 'api.api_proto.users_pb2'
  # @@protoc_insertion_point(class_scope:monorail.GetUserPrefsResponse)
  ))
_sym_db.RegisterMessage(GetUserPrefsResponse)

SetUserPrefsRequest = _reflection.GeneratedProtocolMessageType('SetUserPrefsRequest', (_message.Message,), dict(
  DESCRIPTOR = _SETUSERPREFSREQUEST,
  __module__ = 'api.api_proto.users_pb2'
  # @@protoc_insertion_point(class_scope:monorail.SetUserPrefsRequest)
  ))
_sym_db.RegisterMessage(SetUserPrefsRequest)

SetUserPrefsResponse = _reflection.GeneratedProtocolMessageType('SetUserPrefsResponse', (_message.Message,), dict(
  DESCRIPTOR = _SETUSERPREFSRESPONSE,
  __module__ = 'api.api_proto.users_pb2'
  # @@protoc_insertion_point(class_scope:monorail.SetUserPrefsResponse)
  ))
_sym_db.RegisterMessage(SetUserPrefsResponse)

InviteLinkedParentRequest = _reflection.GeneratedProtocolMessageType('InviteLinkedParentRequest', (_message.Message,), dict(
  DESCRIPTOR = _INVITELINKEDPARENTREQUEST,
  __module__ = 'api.api_proto.users_pb2'
  # @@protoc_insertion_point(class_scope:monorail.InviteLinkedParentRequest)
  ))
_sym_db.RegisterMessage(InviteLinkedParentRequest)

InviteLinkedParentResponse = _reflection.GeneratedProtocolMessageType('InviteLinkedParentResponse', (_message.Message,), dict(
  DESCRIPTOR = _INVITELINKEDPARENTRESPONSE,
  __module__ = 'api.api_proto.users_pb2'
  # @@protoc_insertion_point(class_scope:monorail.InviteLinkedParentResponse)
  ))
_sym_db.RegisterMessage(InviteLinkedParentResponse)

AcceptLinkedChildRequest = _reflection.GeneratedProtocolMessageType('AcceptLinkedChildRequest', (_message.Message,), dict(
  DESCRIPTOR = _ACCEPTLINKEDCHILDREQUEST,
  __module__ = 'api.api_proto.users_pb2'
  # @@protoc_insertion_point(class_scope:monorail.AcceptLinkedChildRequest)
  ))
_sym_db.RegisterMessage(AcceptLinkedChildRequest)

AcceptLinkedChildResponse = _reflection.GeneratedProtocolMessageType('AcceptLinkedChildResponse', (_message.Message,), dict(
  DESCRIPTOR = _ACCEPTLINKEDCHILDRESPONSE,
  __module__ = 'api.api_proto.users_pb2'
  # @@protoc_insertion_point(class_scope:monorail.AcceptLinkedChildResponse)
  ))
_sym_db.RegisterMessage(AcceptLinkedChildResponse)

UnlinkAccountsRequest = _reflection.GeneratedProtocolMessageType('UnlinkAccountsRequest', (_message.Message,), dict(
  DESCRIPTOR = _UNLINKACCOUNTSREQUEST,
  __module__ = 'api.api_proto.users_pb2'
  # @@protoc_insertion_point(class_scope:monorail.UnlinkAccountsRequest)
  ))
_sym_db.RegisterMessage(UnlinkAccountsRequest)

UnlinkAccountsResponse = _reflection.GeneratedProtocolMessageType('UnlinkAccountsResponse', (_message.Message,), dict(
  DESCRIPTOR = _UNLINKACCOUNTSRESPONSE,
  __module__ = 'api.api_proto.users_pb2'
  # @@protoc_insertion_point(class_scope:monorail.UnlinkAccountsResponse)
  ))
_sym_db.RegisterMessage(UnlinkAccountsResponse)



_USERS = _descriptor.ServiceDescriptor(
  name='Users',
  full_name='monorail.Users',
  file=DESCRIPTOR,
  index=0,
  serialized_options=None,
  serialized_start=1397,
  serialized_end=2461,
  methods=[
  _descriptor.MethodDescriptor(
    name='GetUser',
    full_name='monorail.Users.GetUser',
    index=0,
    containing_service=None,
    input_type=_GETUSERREQUEST,
    output_type=api_dot_api__proto_dot_user__objects__pb2._USER,
    serialized_options=None,
  ),
  _descriptor.MethodDescriptor(
    name='ListReferencedUsers',
    full_name='monorail.Users.ListReferencedUsers',
    index=1,
    containing_service=None,
    input_type=_LISTREFERENCEDUSERSREQUEST,
    output_type=_LISTREFERENCEDUSERSRESPONSE,
    serialized_options=None,
  ),
  _descriptor.MethodDescriptor(
    name='GetMemberships',
    full_name='monorail.Users.GetMemberships',
    index=2,
    containing_service=None,
    input_type=_GETMEMBERSHIPSREQUEST,
    output_type=_GETMEMBERSHIPSRESPONSE,
    serialized_options=None,
  ),
  _descriptor.MethodDescriptor(
    name='GetSavedQueries',
    full_name='monorail.Users.GetSavedQueries',
    index=3,
    containing_service=None,
    input_type=_GETSAVEDQUERIESREQUEST,
    output_type=_GETSAVEDQUERIESRESPONSE,
    serialized_options=None,
  ),
  _descriptor.MethodDescriptor(
    name='GetUserStarCount',
    full_name='monorail.Users.GetUserStarCount',
    index=4,
    containing_service=None,
    input_type=_GETUSERSTARCOUNTREQUEST,
    output_type=_GETUSERSTARCOUNTRESPONSE,
    serialized_options=None,
  ),
  _descriptor.MethodDescriptor(
    name='StarUser',
    full_name='monorail.Users.StarUser',
    index=5,
    containing_service=None,
    input_type=_STARUSERREQUEST,
    output_type=_STARUSERRESPONSE,
    serialized_options=None,
  ),
  _descriptor.MethodDescriptor(
    name='GetUserPrefs',
    full_name='monorail.Users.GetUserPrefs',
    index=6,
    containing_service=None,
    input_type=_GETUSERPREFSREQUEST,
    output_type=_GETUSERPREFSRESPONSE,
    serialized_options=None,
  ),
  _descriptor.MethodDescriptor(
    name='SetUserPrefs',
    full_name='monorail.Users.SetUserPrefs',
    index=7,
    containing_service=None,
    input_type=_SETUSERPREFSREQUEST,
    output_type=_SETUSERPREFSRESPONSE,
    serialized_options=None,
  ),
  _descriptor.MethodDescriptor(
    name='SetExpandPermsPreference',
    full_name='monorail.Users.SetExpandPermsPreference',
    index=8,
    containing_service=None,
    input_type=_SETEXPANDPERMSPREFERENCEREQUEST,
    output_type=_SETEXPANDPERMSPREFERENCERESPONSE,
    serialized_options=None,
  ),
  _descriptor.MethodDescriptor(
    name='InviteLinkedParent',
    full_name='monorail.Users.InviteLinkedParent',
    index=9,
    containing_service=None,
    input_type=_INVITELINKEDPARENTREQUEST,
    output_type=_INVITELINKEDPARENTRESPONSE,
    serialized_options=None,
  ),
  _descriptor.MethodDescriptor(
    name='AcceptLinkedChild',
    full_name='monorail.Users.AcceptLinkedChild',
    index=10,
    containing_service=None,
    input_type=_ACCEPTLINKEDCHILDREQUEST,
    output_type=_ACCEPTLINKEDCHILDRESPONSE,
    serialized_options=None,
  ),
  _descriptor.MethodDescriptor(
    name='UnlinkAccounts',
    full_name='monorail.Users.UnlinkAccounts',
    index=11,
    containing_service=None,
    input_type=_UNLINKACCOUNTSREQUEST,
    output_type=_UNLINKACCOUNTSRESPONSE,
    serialized_options=None,
  ),
])
_sym_db.RegisterServiceDescriptor(_USERS)

DESCRIPTOR.services_by_name['Users'] = _USERS

# @@protoc_insertion_point(module_scope)
