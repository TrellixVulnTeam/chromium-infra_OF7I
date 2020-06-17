// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
import {assert} from 'chai';
import {ChopsAnnouncement} from './chops-announcement.js';

let element;

describe('chops-announcement', () => {
  beforeEach(() => {
    element = document.createElement('chops-announcement');
    document.body.appendChild(element);
  });

  afterEach(() => {
    document.body.removeChild(element);
  });

  it('initializes', () => {
    assert.instanceOf(element, ChopsAnnouncement);
  });
});
