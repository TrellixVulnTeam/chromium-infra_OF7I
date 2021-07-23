// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import sinon from 'sinon';
import {fireEvent} from '@testing-library/react';

import {MrEditMetadata} from './mr-edit-metadata.js';
import {ISSUE_EDIT_PERMISSION, ISSUE_EDIT_SUMMARY_PERMISSION,
  ISSUE_EDIT_STATUS_PERMISSION, ISSUE_EDIT_OWNER_PERMISSION,
  ISSUE_EDIT_CC_PERMISSION,
} from 'shared/consts/permissions.js';
import {FIELD_DEF_VALUE_EDIT} from 'reducers/permissions.js';
import {store, resetState} from 'reducers/base.js';

let element;

describe('mr-edit-metadata', () => {
  beforeEach(() => {
    store.dispatch(resetState());
    element = document.createElement('mr-edit-metadata');
    document.body.appendChild(element);

    element.issuePermissions = [ISSUE_EDIT_PERMISSION];

    sinon.stub(store, 'dispatch');
  });

  afterEach(() => {
    document.body.removeChild(element);
    store.dispatch.restore();
  });

  it('initializes', () => {
    assert.instanceOf(element, MrEditMetadata);
  });

  describe('updated sets initial values', () => {
    it('updates owner', async () => {
      element.ownerName = 'goose@bird.org';
      await element.updateComplete;

      assert.equal(element._values.owner, 'goose@bird.org');
    });

    it('updates cc', async () => {
      element.cc = [
        {displayName: 'initial-cc@bird.org', userId: '1234'},
      ];
      await element.updateComplete;

      assert.deepEqual(element._values.cc, ['initial-cc@bird.org']);
    });

    it('updates components', async () => {
      element.components = [{path: 'Hello>World'}];

      await element.updateComplete;

      assert.deepEqual(element._values.components, ['Hello>World']);
    });

    it('updates labels', async () => {
      element.labelNames = ['test-label'];

      await element.updateComplete;

      assert.deepEqual(element._values.labels, ['test-label']);
    });
  });

  describe('saves edit form', () => {
    let saveStub;

    beforeEach(() => {
      saveStub = sinon.stub();
      element.addEventListener('save', saveStub);
    });

    it('saves on form submit', async () => {
      await element.updateComplete;

      element.querySelector('#editForm').dispatchEvent(
          new Event('submit', {bubbles: true, cancelable: true}));

      sinon.assert.calledOnce(saveStub);
    });

    it('saves when clicking the save button', async () => {
      await element.updateComplete;

      element.querySelector('.save-changes').click();

      sinon.assert.calledOnce(saveStub);
    });

    it('does not save on random keydowns', async () => {
      await element.updateComplete;

      element.querySelector('#editForm').dispatchEvent(
          new KeyboardEvent('keydown', {key: 'a', ctrlKey: true}));
      element.querySelector('#editForm').dispatchEvent(
          new KeyboardEvent('keydown', {key: 'b', ctrlKey: false}));
      element.querySelector('#editForm').dispatchEvent(
          new KeyboardEvent('keydown', {key: 'c', metaKey: true}));

      sinon.assert.notCalled(saveStub);
    });

    it('does not save on Enter without Ctrl', async () => {
      await element.updateComplete;

      element.querySelector('#editForm').dispatchEvent(
          new KeyboardEvent('keydown', {key: 'Enter', ctrlKey: false}));

      sinon.assert.notCalled(saveStub);
    });

    it('saves on Ctrl+Enter', async () => {
      await element.updateComplete;

      element.querySelector('#editForm').dispatchEvent(
          new KeyboardEvent('keydown', {key: 'Enter', ctrlKey: true}));

      sinon.assert.calledOnce(saveStub);
    });

    it('saves on Ctrl+Meta', async () => {
      await element.updateComplete;

      element.querySelector('#editForm').dispatchEvent(
          new KeyboardEvent('keydown', {key: 'Enter', metaKey: true}));

      sinon.assert.calledOnce(saveStub);
    });
  });

  it('disconnecting element reports form is not dirty', () => {
    element.formName = 'test';

    assert.isFalse(store.dispatch.calledOnce);

    document.body.removeChild(element);

    assert.isTrue(store.dispatch.calledOnce);
    sinon.assert.calledWith(
        store.dispatch,
        {
          type: 'REPORT_DIRTY_FORM',
          name: 'test',
          isDirty: false,
        },
    );

    document.body.appendChild(element);
  });

  it('_processChanges fires change event', async () => {
    await element.updateComplete;

    const changeStub = sinon.stub();
    element.addEventListener('change', changeStub);

    element._processChanges();

    sinon.assert.calledOnce(changeStub);
  });

  it('save button disabled when disabled is true', async () => {
    // Check that save button is initially disabled.
    await element.updateComplete;

    // Wait for <chops-chip-input> to finish its update cycle.
    await element.updateComplete;

    const button = element.querySelector('.save-changes');

    assert.isTrue(element.disabled);
    assert.isTrue(button.disabled);

    element.isDirty = true;

    await element.updateComplete;

    assert.isFalse(element.disabled);
    assert.isFalse(button.disabled);
  });

  it('editing form sets isDirty to true or false', async () => {
    await element.updateComplete;

    assert.isFalse(element.isDirty);

    // User makes some changes.
    const comment = element.querySelector('#commentText');
    comment.value = 'Value';
    comment.dispatchEvent(new Event('keyup'));

    assert.isTrue(element.isDirty);

    // User undoes the changes.
    comment.value = '';
    comment.dispatchEvent(new Event('keyup'));

    assert.isFalse(element.isDirty);
  });

  it('reseting form disables save button', async () => {
    // Check that save button is initially disabled.
    assert.isTrue(element.disabled);

    // User makes some changes.
    element.isDirty = true;

    // Check that save button is not disabled.
    assert.isFalse(element.disabled);

    // Reset form.
    await element.updateComplete;
    await element.reset();

    // Check that save button is still disabled.
    assert.isTrue(element.disabled);
  });

  it('save button is enabled if request fails', async () => {
    // Check that save button is initially disabled.
    assert.isTrue(element.disabled);

    // User makes some changes.
    element.isDirty = true;

    // Check that save button is not disabled.
    assert.isFalse(element.disabled);

    // User submits the change.
    element.saving = true;

    // Check that save button is disabled.
    assert.isTrue(element.disabled);

    // Request fails.
    element.saving = false;
    element.error = 'error';

    // Check that save button is re-enabled.
    assert.isFalse(element.disabled);
  });

  it('delta empty when no changes', async () => {
    await element.updateComplete;
    assert.deepEqual(element.delta, {});
  });

  it('toggling checkbox toggles sendEmail', async () => {
    element.sendEmail = false;

    await element.updateComplete;
    const checkbox = element.querySelector('#sendEmail');

    await checkbox.updateComplete;

    checkbox.click();
    await element.updateComplete;

    assert.equal(checkbox.checked, true);
    assert.equal(element.sendEmail, true);

    checkbox.click();
    await element.updateComplete;

    assert.equal(checkbox.checked, false);
    assert.equal(element.sendEmail, false);

    checkbox.click();
    await element.updateComplete;

    assert.equal(checkbox.checked, true);
    assert.equal(element.sendEmail, true);
  });

  it('changing status produces delta change (lit-element)', async () => {
    element.statuses = [
      {'status': 'New'},
      {'status': 'Old'},
      {'status': 'Test'},
    ];
    element.status = 'New';

    await element.updateComplete;

    const statusComponent = element.querySelector('#statusInput');
    statusComponent.status = 'Old';

    await element.updateComplete;

    assert.deepEqual(element.delta, {
      status: 'Old',
    });
  });

  it('changing owner produces delta change (React)', async () => {
    element.ownerName = 'initial-owner@bird.org';
    await element.updateComplete;

    const input = element.querySelector('#ownerInput');
    await enterInput(element, input, 'new-owner@bird.org');

    const expected = {ownerRef: {displayName: 'new-owner@bird.org'}};
    assert.deepEqual(element.delta, expected);
  });

  it('adding CC produces delta change (React)', async () => {
    element.cc = [
      {displayName: 'initial-cc@bird.org', userId: '1234'},
    ];

    await element.updateComplete;

    const input = element.querySelector('#ccInput');
    await enterInput(element, input, 'another@bird.org');

    await element.updateComplete;

    const expected = {
      ccRefsAdd: [{displayName: 'another@bird.org'}],
      ccRefsRemove: [{displayName: 'initial-cc@bird.org'}],
    };
    assert.deepEqual(element.delta, expected);
  });

  it('invalid status throws', async () => {
    element.statuses = [
      {'status': 'New'},
      {'status': 'Old'},
      {'status': 'Duplicate'},
    ];
    element.status = 'Duplicate';

    await element.updateComplete;

    const statusComponent = element.querySelector('#statusInput');
    statusComponent.shadowRoot.querySelector('#mergedIntoInput').value = 'xx';
    assert.deepEqual(element.delta, {});
    assert.equal(
        element.error,
        'Invalid issue ref: xx. Expected [projectName:]issueId.');
  });

  it('cannot block an issue on itself', async () => {
    element.projectName = 'proj';
    element.issueRef = {projectName: 'proj', localId: 123};

    await element.updateComplete;

    for (const fieldName of ['blockedOn', 'blocking']) {
      const input =
        element.querySelector(`#${fieldName}Input`);
      await enterInput(element, input, '123');
      assert.deepEqual(element.delta, {});
      assert.equal(
          element.error,
          `Invalid issue ref: 123. Cannot merge or block an issue on itself.`);
      fireEvent.keyDown(input, {key: 'Backspace', code: 'Backspace'});
      await element.updateComplete;

      await enterInput(element, input, 'proj:123');
      assert.deepEqual(element.delta, {});
      assert.equal(
          element.error,
          `Invalid issue ref: proj:123. ` +
        'Cannot merge or block an issue on itself.');
      fireEvent.keyDown(input, {key: 'Backspace', code: 'Backspace'});
      await element.updateComplete;

      await enterInput(element, input, 'proj2:123');
      assert.notDeepEqual(element.delta, {});
      assert.equal(element.error, '');
      fireEvent.keyDown(input, {key: 'Backspace', code: 'Backspace'});
      await element.updateComplete;
    }
  });

  it('cannot merge an issue into itself', async () => {
    element.statuses = [
      {'status': 'New'},
      {'status': 'Duplicate'},
    ];
    element.status = 'New';
    element.projectName = 'proj';
    element.issueRef = {projectName: 'proj', localId: 123};

    await element.updateComplete;

    const statusComponent = element.querySelector('#statusInput');
    const root = statusComponent.shadowRoot;
    const statusInput = root.querySelector('#statusInput');
    statusInput.value = 'Duplicate';
    statusInput.dispatchEvent(new Event('change'));

    await element.updateComplete;

    root.querySelector('#mergedIntoInput').value = 'proj:123';
    assert.deepEqual(element.delta, {});
    assert.equal(
        element.error,
        `Invalid issue ref: proj:123. Cannot merge or block an issue on itself.`);

    root.querySelector('#mergedIntoInput').value = '123';
    assert.deepEqual(element.delta, {});
    assert.equal(
        element.error,
        `Invalid issue ref: 123. Cannot merge or block an issue on itself.`);

    root.querySelector('#mergedIntoInput').value = 'proj2:123';
    assert.notDeepEqual(element.delta, {});
    assert.equal(element.error, '');
  });

  it('cannot set invalid emails', async () => {
    await element.updateComplete;

    const ccInput = element.querySelector('#ccInput');
    await enterInput(element, ccInput, 'invalid!email');

    assert.deepEqual(element.delta, {});
    assert.equal(
        element.error,
        `Invalid email address: invalid!email`);

    const input = element.querySelector('#ownerInput');
    await enterInput(element, input, 'invalid!email2');

    assert.deepEqual(element.delta, {});
    assert.equal(
        element.error,
        `Invalid email address: invalid!email2`);
  });

  it('can remove invalid values', async () => {
    element.projectName = 'proj';
    element.issueRef = {projectName: 'proj', localId: 123};

    element.statuses = [
      {'status': 'Duplicate'},
    ];
    element.status = 'Duplicate';
    element.mergedInto = element.issueRef;

    element.blockedOn = [element.issueRef];
    element.blocking = [element.issueRef];

    await element.updateComplete;

    const blockedOnInput = element.querySelector('#blockedOnInput');
    const blockingInput = element.querySelector('#blockingInput');
    const statusInput = element.querySelector('#statusInput');

    await element.updateComplete;

    const mergedIntoInput =
      statusInput.shadowRoot.querySelector('#mergedIntoInput');

    fireEvent.keyDown(blockedOnInput, {key: 'Backspace', code: 'Backspace'});
    await element.updateComplete;
    fireEvent.keyDown(blockingInput, {key: 'Backspace', code: 'Backspace'});
    await element.updateComplete;
    mergedIntoInput.value = 'proj:124';
    await element.updateComplete;

    assert.deepEqual(
        element.delta,
        {
          blockedOnRefsRemove: [{projectName: 'proj', localId: 123}],
          blockingRefsRemove: [{projectName: 'proj', localId: 123}],
          mergedIntoRef: {projectName: 'proj', localId: 124},
        });
    assert.equal(element.error, '');
  });

  it('not changing status produces no delta', async () => {
    element.statuses = [
      {'status': 'Duplicate'},
    ];
    element.status = 'Duplicate';

    element.mergedInto = {
      projectName: 'chromium',
      localId: 1234,
    };

    element.projectName = 'chromium';

    await element.updateComplete;
    await element.updateComplete; // Merged input updates its value.

    assert.deepEqual(element.delta, {});
  });

  it('changing status to duplicate produces delta change', async () => {
    element.statuses = [
      {'status': 'New'},
      {'status': 'Duplicate'},
    ];
    element.status = 'New';

    await element.updateComplete;

    const statusComponent = element.querySelector(
        '#statusInput');
    const root = statusComponent.shadowRoot;
    const statusInput = root.querySelector('#statusInput');
    statusInput.value = 'Duplicate';
    statusInput.dispatchEvent(new Event('change'));

    await element.updateComplete;

    root.querySelector('#mergedIntoInput').value = 'chromium:1234';
    assert.deepEqual(element.delta, {
      status: 'Duplicate',
      mergedIntoRef: {
        projectName: 'chromium',
        localId: 1234,
      },
    });
  });

  it('changing summary produces delta change', async () => {
    element.summary = 'Old summary';

    await element.updateComplete;

    element.querySelector(
        '#summaryInput').value = 'newfangled fancy summary';
    assert.deepEqual(element.delta, {
      summary: 'newfangled fancy summary',
    });
  });

  it('custom fields the user cannot edit should be hidden', async () => {
    element.projectName = 'proj';
    const fieldName = 'projects/proj/fieldDefs/1';
    const restrictedFieldName = 'projects/proj/fieldDefs/2';
    element._permissions = {
      [fieldName]: {permissions: [FIELD_DEF_VALUE_EDIT]},
      [restrictedFieldName]: {permissions: []}};
    element.fieldDefs = [
      {
        fieldRef: {
          fieldName: 'normalFd',
          fieldId: 1,
          type: 'ENUM_TYPE',
        },
      },
      {
        fieldRef: {
          fieldName: 'cantEditFd',
          fieldId: 2,
          type: 'ENUM_TYPE',
        },
      },
    ];

    await element.updateComplete;
    assert.isFalse(element.querySelector('#normalFdInput').hidden);
    assert.isTrue(element.querySelector('#cantEditFdInput').hidden);
  });

  it('changing custom fields produces delta', async () => {
    element.fieldValueMap = new Map([['fakefield', ['prev value']]]);
    element.fieldDefs = [
      {
        fieldRef: {
          fieldName: 'testField',
          fieldId: 1,
          type: 'ENUM_TYPE',
        },
      },
      {
        fieldRef: {
          fieldName: 'fakeField',
          fieldId: 2,
          type: 'ENUM_TYPE',
        },
      },
    ];

    await element.updateComplete;

    element.querySelector('#testFieldInput').setValue('test value');
    element.querySelector('#fakeFieldInput').setValue('');
    assert.deepEqual(element.delta, {
      fieldValsAdd: [
        {
          fieldRef: {
            fieldName: 'testField',
            fieldId: 1,
            type: 'ENUM_TYPE',
          },
          value: 'test value',
        },
      ],
      fieldValsRemove: [
        {
          fieldRef: {
            fieldName: 'fakeField',
            fieldId: 2,
            type: 'ENUM_TYPE',
          },
          value: 'prev value',
        },
      ],
    });
  });

  it('changing approvers produces delta', async () => {
    element.isApproval = true;
    element.hasApproverPrivileges = true;
    element.approvers = [
      {displayName: 'foo@example.com', userId: '1'},
      {displayName: 'bar@example.com', userId: '2'},
      {displayName: 'baz@example.com', userId: '3'},
    ];

    await element.updateComplete;
    await element.updateComplete;

    element.querySelector('#approversInput').setValue(
        ['chicken@example.com', 'foo@example.com', 'dog@example.com']);

    await element.updateComplete;

    assert.deepEqual(element.delta, {
      approverRefsAdd: [
        {displayName: 'chicken@example.com'},
        {displayName: 'dog@example.com'},
      ],
      approverRefsRemove: [
        {displayName: 'bar@example.com'},
        {displayName: 'baz@example.com'},
      ],
    });
  });

  it('changing blockedon produces delta change (React)', async () => {
    element.blockedOn = [
      {projectName: 'chromium', localId: '1234'},
      {projectName: 'monorail', localId: '4567'},
    ];
    element.projectName = 'chromium';

    await element.updateComplete;
    await element.updateComplete;

    const input = element.querySelector('#blockedOnInput');

    fireEvent.keyDown(input, {key: 'Backspace', code: 'Backspace'});
    await element.updateComplete;

    await enterInput(element, input, 'v8:5678');

    assert.deepEqual(element.delta, {
      blockedOnRefsAdd: [{
        projectName: 'v8',
        localId: 5678,
      }],
      blockedOnRefsRemove: [{
        projectName: 'monorail',
        localId: 4567,
      }],
    });
  });

  it('_optionsForField computes options', () => {
    const optionsPerEnumField = new Map([
      ['enumfield', [{optionName: 'one'}, {optionName: 'two'}]],
    ]);
    assert.deepEqual(
        element._optionsForField(optionsPerEnumField, new Map(), 'enumField'), [
          {
            optionName: 'one',
          },
          {
            optionName: 'two',
          },
        ]);
  });

  it('changing enum fields produces delta', async () => {
    element.fieldDefs = [
      {
        fieldRef: {
          fieldName: 'enumField',
          fieldId: 1,
          type: 'ENUM_TYPE',
        },
        isMultivalued: true,
      },
    ];

    element.optionsPerEnumField = new Map([
      ['enumfield', [{optionName: 'one'}, {optionName: 'two'}]],
    ]);

    await element.updateComplete;
    await element.updateComplete;

    element.querySelector(
        '#enumFieldInput').setValue(['one', 'two']);

    await element.updateComplete;

    assert.deepEqual(element.delta, {
      fieldValsAdd: [
        {
          fieldRef: {
            fieldName: 'enumField',
            fieldId: 1,
            type: 'ENUM_TYPE',
          },
          value: 'one',
        },
        {
          fieldRef: {
            fieldName: 'enumField',
            fieldId: 1,
            type: 'ENUM_TYPE',
          },
          value: 'two',
        },
      ],
    });
  });

  it('changing multiple single valued enum fields', async () => {
    element.fieldDefs = [
      {
        fieldRef: {
          fieldName: 'enumField',
          fieldId: 1,
          type: 'ENUM_TYPE',
        },
      },
      {
        fieldRef: {
          fieldName: 'enumField2',
          fieldId: 2,
          type: 'ENUM_TYPE',
        },
      },
    ];

    element.optionsPerEnumField = new Map([
      ['enumfield', [{optionName: 'one'}, {optionName: 'two'}]],
      ['enumfield2', [{optionName: 'three'}, {optionName: 'four'}]],
    ]);

    await element.updateComplete;

    element.querySelector('#enumFieldInput').setValue(['two']);
    element.querySelector('#enumField2Input').setValue(['three']);

    await element.updateComplete;

    assert.deepEqual(element.delta, {
      fieldValsAdd: [
        {
          fieldRef: {
            fieldName: 'enumField',
            fieldId: 1,
            type: 'ENUM_TYPE',
          },
          value: 'two',
        },
        {
          fieldRef: {
            fieldName: 'enumField2',
            fieldId: 2,
            type: 'ENUM_TYPE',
          },
          value: 'three',
        },
      ],
    });
  });

  it('adding components produces delta', async () => {
    await element.updateComplete;

    element.isApproval = false;
    element.issuePermissions = [ISSUE_EDIT_PERMISSION];

    element.components = [];

    await element.updateComplete;

    element._values.components = ['Hello>World'];

    await element.updateComplete;

    assert.deepEqual(element.delta, {
      compRefsAdd: [
        {path: 'Hello>World'},
      ],
    });

    element._values.components = ['Hello>World', 'Test', 'Multi'];

    await element.updateComplete;

    assert.deepEqual(element.delta, {
      compRefsAdd: [
        {path: 'Hello>World'},
        {path: 'Test'},
        {path: 'Multi'},
      ],
    });

    element._values.components = [];

    await element.updateComplete;

    assert.deepEqual(element.delta, {});
  });

  it('removing components produces delta', async () => {
    await element.updateComplete;

    element.isApproval = false;
    element.issuePermissions = [ISSUE_EDIT_PERMISSION];

    element.components = [{path: 'Hello>World'}];

    await element.updateComplete;

    element._values.components = [];

    await element.updateComplete;

    assert.deepEqual(element.delta, {
      compRefsRemove: [
        {path: 'Hello>World'},
      ],
    });
  });

  it('approver input appears when user has privileges', async () => {
    assert.isNull(element.querySelector('#approversInput'));
    element.isApproval = true;
    element.hasApproverPrivileges = true;

    await element.updateComplete;

    assert.isNotNull(element.querySelector('#approversInput'));
  });

  it('reset sets controlled values to default', async () => {
    element.ownerName = 'burb@bird.com';
    element.cc = [
      {displayName: 'flamingo@bird.com', userId: '1234'},
      {displayName: 'penguin@bird.com', userId: '5678'},
    ];
    element.components = [{path: 'Bird>Penguin'}];
    element.labelNames = ['chickadee-chirp'];
    element.blockedOn = [{localId: 1234, projectName: 'project'}];
    element.blocking = [{localId: 5678, projectName: 'other-project'}];
    element.projectName = 'project';

    // Update cycle is needed because <mr-edit-metadata> initializes
    // this.values in updated().
    await element.updateComplete;

    const initialValues = {
      owner: 'burb@bird.com',
      cc: ['flamingo@bird.com', 'penguin@bird.com'],
      components: ['Bird>Penguin'],
      labels: ['chickadee-chirp'],
      blockedOn: ['1234'],
      blocking: ['other-project:5678'],
    };

    assert.deepEqual(element._values, initialValues);

    element._values = {
      owner: 'newburb@hello.com',
      cc: ['noburbs@wings.com'],
    };
    element.reset();

    assert.deepEqual(element._values, initialValues);
  })

  it('reset empties form values', async () => {
    element.fieldDefs = [
      {
        fieldRef: {
          fieldName: 'testField',
          fieldId: 1,
          type: 'ENUM_TYPE',
        },
      },
      {
        fieldRef: {
          fieldName: 'fakeField',
          fieldId: 2,
          type: 'ENUM_TYPE',
        },
      },
    ];

    await element.updateComplete;

    const uploader = element.querySelector('mr-upload');
    uploader.files = [
      {name: 'test.png'},
      {name: 'rutabaga.png'},
    ];

    element.querySelector('#testFieldInput').setValue('testy test');
    element.querySelector('#fakeFieldInput').setValue('hello world');

    await element.reset();

    assert.lengthOf(element.querySelector('#testFieldInput').value, 0);
    assert.lengthOf(element.querySelector('#fakeFieldInput').value, 0);
    assert.lengthOf(uploader.files, 0);
  });

  it('reset results in empty delta', async () => {
    element.ownerName = 'goose@bird.org';
    await element.updateComplete;

    element._values.owner = 'penguin@bird.org';
    const expected = {ownerRef: {displayName: 'penguin@bird.org'}};
    assert.deepEqual(element.delta, expected);

    await element.reset();
    assert.deepEqual(element.delta, {});
  });

  it('edit issue permissions', async () => {
    const allFields = ['summary', 'status', 'owner', 'cc'];
    const testCases = [
      {permissions: [], nonNull: []},
      {permissions: [ISSUE_EDIT_PERMISSION], nonNull: allFields},
      {permissions: [ISSUE_EDIT_SUMMARY_PERMISSION], nonNull: ['summary']},
      {permissions: [ISSUE_EDIT_STATUS_PERMISSION], nonNull: ['status']},
      {permissions: [ISSUE_EDIT_OWNER_PERMISSION], nonNull: ['owner']},
      {permissions: [ISSUE_EDIT_CC_PERMISSION], nonNull: ['cc']},
    ];
    element.statuses = [{'status': 'Foo'}];

    for (const testCase of testCases) {
      element.issuePermissions = testCase.permissions;
      await element.updateComplete;

      allFields.forEach((fieldName) => {
        const field = element.querySelector(`#${fieldName}Input`);
        if (testCase.nonNull.includes(fieldName)) {
          assert.isNotNull(field);
        } else {
          assert.isNull(field);
        }
      });
    }
  });

  it('duplicate issue is rendered correctly', async () => {
    element.statuses = [
      {'status': 'Duplicate'},
    ];
    element.status = 'Duplicate';
    element.projectName = 'chromium';
    element.mergedInto = {
      projectName: 'chromium',
      localId: 1234,
    };

    await element.updateComplete;
    await element.updateComplete;

    const statusComponent = element.querySelector('#statusInput');
    const root = statusComponent.shadowRoot;
    assert.equal(
        root.querySelector('#mergedIntoInput').value, '1234');
  });

  it('duplicate issue on different project is rendered correctly', async () => {
    element.statuses = [
      {'status': 'Duplicate'},
    ];
    element.status = 'Duplicate';
    element.projectName = 'chromium';
    element.mergedInto = {
      projectName: 'monorail',
      localId: 1234,
    };

    await element.updateComplete;
    await element.updateComplete;

    const statusComponent = element.querySelector('#statusInput');
    const root = statusComponent.shadowRoot;
    assert.equal(
        root.querySelector('#mergedIntoInput').value, 'monorail:1234');
  });

  it('filter out deleted users', async () => {
    element.cc = [
      {displayName: 'test@example.com', userId: '1234'},
      {displayName: 'a_deleted_user'},
      {displayName: 'someone@example.com', userId: '5678'},
    ];

    await element.updateComplete;

    assert.deepEqual(element._values.cc, [
      'test@example.com',
      'someone@example.com',
    ]);
  });

  it('renders valid markdown description with preview', async () => {
    await element.updateComplete;

    element.prefs = new Map([['render_markdown', true]]);
    element.projectName = 'monkeyrail';
    sinon.stub(element, 'getCommentContent').returns('# h1');

    await element.updateComplete;

    assert.isTrue(element._renderMarkdown);

    const previewMarkdown = element.querySelector('.markdown-preview');
    assert.isNotNull(previewMarkdown);

    const headerText = previewMarkdown.querySelector('h1').textContent;
    assert.equal(headerText, 'h1');
  });

  it('does not show preview when markdown is disabled', async () => {
    element.prefs = new Map([['render_markdown', false]]);
    element.projectName = 'monkeyrail';
    sinon.stub(element, 'getCommentContent').returns('# h1');

    await element.updateComplete;

    const previewMarkdown = element.querySelector('.markdown-preview');
    assert.isNull(previewMarkdown);
  });

  it('does not show preview when no input', async () => {
    element.prefs = new Map([['render_markdown', true]]);
    element.projectName = 'monkeyrail';
    sinon.stub(element, 'getCommentContent').returns('');

    await element.updateComplete;

    const previewMarkdown = element.querySelector('.markdown-preview');
    assert.isNull(previewMarkdown);
  });
});

/**
 * Types text into an input field and presses Enter.
 * @param {MrEditMetadata} element The component that controls the input field.
 * @param {HTMLInputElement} input The input field to enter text in.
 * @param {string} value The text to enter in the input field.
 */
async function enterInput(element, input, value) {
  fireEvent.change(input, {target: {value}});
  fireEvent.keyDown(input, {key: 'Enter', code: 'Enter'});
  await element.updateComplete;
}
