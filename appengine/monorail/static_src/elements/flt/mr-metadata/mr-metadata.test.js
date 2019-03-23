// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import {MrMetadata} from './mr-metadata.js';
import {actionType} from '../../redux/redux-mixin.js';


let element;

suite('mr-metadata', () => {
  setup(() => {
    element = document.createElement('mr-metadata');
    document.body.appendChild(element);

    element.projectName = 'proj';
  });

  teardown(() => {
    document.body.removeChild(element);
    element.dispatchAction({type: actionType.RESET_STATE});
  });

  test('initializes', () => {
    assert.instanceOf(element, MrMetadata);
  });
});
