# Generated by the pRPC protocol buffer compiler plugin.  DO NOT EDIT!
# source: api/api_proto/projects.proto

import base64
import zlib

from google.protobuf import descriptor_pb2

# Includes description of the api/api_proto/projects.proto and all of its transitive
# dependencies. Includes source code info.
FILE_DESCRIPTOR_SET = descriptor_pb2.FileDescriptorSet()
FILE_DESCRIPTOR_SET.ParseFromString(zlib.decompress(base64.b64decode(
    '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')))
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
