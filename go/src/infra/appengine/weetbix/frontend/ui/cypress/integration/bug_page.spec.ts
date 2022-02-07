// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
import { setupTestRule } from './test_data';

describe('Bug Page', () => {
    beforeEach(() => {
        // Login.
        cy.visit('/').contains('LOGIN').click();

        setupTestRule();
    })

    it('redirects to rule', () => {
        cy.visit('/b/chromium/123');
        cy.get('rule-section').get('[data-cy=rule-definition]').contains('test = "cypress test 1"');
        cy.get('rule-section').get('[data-cy=rule-enabled]').contains('Yes');
    })

    it('no matching rule exists', () => {
        cy.visit('/b/chromium/404');
        cy.get('bug-page').contains('No rule found matching the specified bug (monorail:chromium/404).');
    })
})
