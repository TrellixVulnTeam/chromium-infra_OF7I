// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

describe('New Rule Page', () => {
    beforeEach(() => {
        // Login.
        cy.visit('/').get('button').click();
    })
    it('create rule from scratch', () => {
        cy.visit('/projects/chromium/rules/new');

        cy.get('new-rule-page').get('[data-cy=bug-system-dropdown]').contains('crbug.com');
        cy.get('new-rule-page').get('[data-cy=bug-number-textbox]').get('[type=text]').type('{selectall}100');
        cy.get('new-rule-page').get('[data-cy=rule-definition-textbox]').get('textarea').type('{selectall}test = "create test 1"');

        cy.intercept('POST', '/api/projects/chromium/rules', (req) => {
            let requestBody = req.body;
            assert.isNotEmpty(requestBody.xsrfToken);
            assert.strictEqual(requestBody.rule.ruleDefinition, 'test = "create test 1"');
            assert.deepEqual(requestBody.rule.bugId, { system: 'monorail', id: 'chromium/100' });
            assert.deepEqual(requestBody.rule.sourceCluster, { algorithm: '', id: '' });

            req.reply({
                project: 'chromium',
                // This is a real rule that exists in the dev database, the
                // same used for rule section UI tests.
                ruleId: '9e5a795d961af2adb52d92f337e6ed2f',
            })
        }).as('createRule');

        cy.get('new-rule-page').get('[data-cy=create-button]').click();
        cy.wait('@createRule');

        cy.get('body').contains('9e5a795d961af2adb52d92f337e6ed2f');
    })
    it('create rule from cluster', () => {
        let rule = 'test = "create test 2"';
        cy.visit(`/projects/chromium/rules/new?rule=${encodeURIComponent(rule)}&sourceAlg=reason-v1&sourceId=1234567890abcedf1234567890abcedf`);

        cy.get('new-rule-page').get('[data-cy=bug-system-dropdown]').contains('crbug.com');
        cy.get('new-rule-page').get('[data-cy=bug-number-textbox]').get('[type=text]').type('{selectall}100');

        cy.intercept('POST', '/api/projects/chromium/rules', (req) => {
            let requestBody = req.body;
            assert.isNotEmpty(requestBody.xsrfToken);
            assert.strictEqual(requestBody.rule.ruleDefinition, 'test = "create test 2"');
            assert.deepEqual(requestBody.rule.bugId, { system: 'monorail', id: 'chromium/100' });
            assert.deepEqual(requestBody.rule.sourceCluster, { algorithm: 'reason-v1', id: '1234567890abcedf1234567890abcedf' });

            req.reply({
                project: 'chromium',
                // This is a real rule that exists in the dev database, the
                // same used for rule section UI tests.
                ruleId: '9e5a795d961af2adb52d92f337e6ed2f',
            })
        }).as('createRule');

        cy.get('new-rule-page').get('[data-cy=create-button]').click();
        cy.wait('@createRule');

        cy.get('body').contains('9e5a795d961af2adb52d92f337e6ed2f');
    })
    it('displays validation errors', () => {
        cy.visit('/projects/chromium/rules/new');
        cy.get('new-rule-page').get('[data-cy=bug-system-dropdown]').contains('crbug.com');
        cy.get('new-rule-page').get('[data-cy=bug-number-textbox]').get('[type=text]').type('{selectall}100');
        cy.get('new-rule-page').get('[data-cy=rule-definition-textbox]').get('textarea').type('{selectall}test = INVALID');

        cy.get('new-rule-page').get('[data-cy=create-button]').click();

        cy.get('body').contains('Validation error: rule definition is not valid: undeclared identifier "invalid".');
    })
})
