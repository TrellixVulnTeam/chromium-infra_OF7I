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

    it('redirects if single matching rule found', () => {
        cy.visit('/b/chromium/920867');
        cy.get('rule-section').get('[data-cy=bug]').contains('crbug.com/920867');
    })

    it('no matching rule exists', () => {
        cy.visit('/b/chromium/404');
        cy.get('bug-page').contains('No rule found matching the specified bug (monorail:chromium/404).');
    })

    it('multiple matching rules found', () => {
        cy.intercept('POST', '/prpc/weetbix.v1.Rules/LookupBug', (req) => {
            const requestBody = req.body;
            assert.deepEqual(requestBody, { system: 'monorail', id: 'chromium/1234' });

            const response = {
                // This is a real rule that exists in the dev database, the
                // same used for rule section UI tests.
                rules: [
                    'projects/chromium/rules/ac856b1827dc1cb845486edbf4b80cfa',
                    'projects/chromiumos/rules/1234567890abcedf1234567890abcdef',
                ],
            }
            // Construct pRPC response.
            const body = ")]}'" + JSON.stringify(response);
            req.reply(body, {
                "X-Prpc-Grpc-Code": "0",
            })
        }).as('lookupBug');

        cy.visit('/b/chromium/1234');
        cy.wait('@lookupBug');

        cy.get('body').contains('chromiumos');
        cy.get('body').contains('chromium').click();

        cy.get('rule-section').get('[data-cy=rule-definition]').contains('test = "cypress test 1"');
    })
})
