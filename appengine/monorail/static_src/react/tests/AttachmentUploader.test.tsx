// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React from 'react';
import {assert} from 'chai';
import {cleanup, render} from '@testing-library/react';
import AttachmentUploader from 'react/issue-wizard/AttachmentUploader.tsx';

describe('IssueWizard Attachment Uploader', () => {
  afterEach(cleanup);

  it('render', () => {
    render(<AttachmentUploader files={[]} setFiles={()=>{}}/>)
    const uploadButton = document.getElementById('file-uploader');
    assert.isNotNull(uploadButton);
  });

  it('render files name', () => {
    const files = [
      {name: '1.txt'},
      {name: '2.txt'},
      {name: '3.txt'},
    ];
    render(<AttachmentUploader files={files} setFiles={()=>{}}/>)
    const items = document.querySelectorAll('li');
    assert.equal(items.length, 3);

    assert.include(items[0].textContent, '1.txt');
    assert.include(items[1].textContent, '2.txt');
    assert.include(items[2].textContent, '3.txt');
  });

  it('remove files', () => {
    let files = [
      {name: '1.txt'},
      {name: '2.txt'},
      {name: '3.txt'},
    ];
    render(<AttachmentUploader files={files} setFiles={(f: Array<any>)=>{files = f;}} setSubmitEnable={()=>{}}/>)
    const items = document.querySelectorAll('li');
    assert.equal(items.length, 3);

    const removeButton = items[1].querySelector('button');
    assert.isNotNull(removeButton);

    removeButton?.click();
    assert.equal(files.length, 2);
    assert.equal(files[0].name, '1.txt');
    assert.equal(files[1].name, '3.txt');
  })
});
