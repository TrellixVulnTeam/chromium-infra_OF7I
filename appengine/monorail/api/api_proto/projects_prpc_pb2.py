# Generated by the pRPC protocol buffer compiler plugin.  DO NOT EDIT!
# source: api/api_proto/projects.proto

import base64
import zlib

from google.protobuf import descriptor_pb2

# Includes description of the api/api_proto/projects.proto and all of its transitive
# dependencies. Includes source code info.
FILE_DESCRIPTOR_SET = descriptor_pb2.FileDescriptorSet()
FILE_DESCRIPTOR_SET.ParseFromString(zlib.decompress(base64.b64decode(
    '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'
    'I+jP0fvN6vPA==')))
_INDEX = {
    f.name: {
      'descriptor': f,
      'services': {s.name: s for s in f.service},
    }
    for f in FILE_DESCRIPTOR_SET.file
}


ProjectsServiceDescription = {
  'file_descriptor_set': FILE_DESCRIPTOR_SET,
  'file_descriptor': _INDEX[u'api/api_proto/projects.proto']['descriptor'],
  'service_descriptor': _INDEX[u'api/api_proto/projects.proto']['services'][u'Projects'],
}
