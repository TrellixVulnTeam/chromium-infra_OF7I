# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import kfp
from kfp import dsl
from kfp import components

# This is a single component pipeline that works as a base
# implementation. It does not have input parameters as the
# current implementation has hardcoded query parameters.
# This will need to be changed when updating for multi-step
# pipeline.


@kfp.dsl.component
def full_LSTM_component():
    return kfp.dsl.ContainerOp(name='LSTM component',
                               image='gcr.io/chromeperf-datalab/lstm_v2')


@dsl.pipeline(name='LSTM pipeline',
              description='LSTM anomaly detection pipeline')
def lstm_pipeline():
    full_LSTM_component()


if __name__ == '__main__':
    # Compiling the pipeline
    kfp.compiler.Compiler().compile(lstm_pipeline, __file__ + '.zip')
