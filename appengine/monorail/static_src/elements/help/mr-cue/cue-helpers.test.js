// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import {cueNameToSpec, specToCueName} from './cue-helpers.js';


describe('cue-helpers', () => {
  describe('cueNameToSpec', () => {
    it('appends cue prefix', () => {
      assert.equal(cueNameToSpec('test'), 'cue.test');
    });
  });

  describe('specToCueName', () => {
    it('extracts cue name from matching spec', () => {
      assert.equal(specToCueName('cue.test'), 'test');
      assert.equal(specToCueName('cue.hello-world'), 'hello-world');
      assert.equal(specToCueName('cue.under_score'), 'under_score');
    });

    it('does not extract cue name from non-matching spec', () => {
      assert.equal(specToCueName('.cue.test'), '');
      assert.equal(specToCueName('hello-world-cue.'), '');
      assert.equal(specToCueName('cu.under_score'), '');
      assert.equal(specToCueName('field'), '');
    });
  });
});
