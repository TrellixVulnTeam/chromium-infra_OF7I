# Copyright 2022 The Chromium Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

PYTHON_VERSION_COMPATIBILITY = "PY2+3"

DEPS = [
  'snoopy',
  'recipe_engine/path',
]

def RunSteps(api):
  # Report task stage.
  api.snoopy.report_stage("start")
  # Report another stage; the module shouldn't install snoopy_broker again.
  api.snoopy.report_stage("fetch", snoopy_url="http://test.local")

  # Report cipd digest.
  api.snoopy.report_cipd("deadbeef", "example/cipd/package", "fakeiid",
                         snoopy_url="http://test.local")

  # Report gcs artifact digest.
  api.snoopy.report_gcs("deadbeef", "gs://bucket/path/to/binary",
                        snoopy_url="http://test.local")


def GenTests(api):
  yield api.test('simple') + api.snoopy(54321)
