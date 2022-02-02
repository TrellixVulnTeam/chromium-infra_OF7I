// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

describe('Rule Section', () => {
    beforeEach(() => {
        // Login.
        cy.visit('/').get('button').click();

        cy.request({
            url: '/api/authState',
            headers: {
                'Sec-Fetch-Site': 'same-origin',
            }
        }).then((response) => {
            assert.strictEqual(response.status, 200);
            const body = response.body;
            const accessToken = body.accessToken;
            assert.isString(accessToken);
            assert.notEqual(accessToken, '');

            // Set initial rule state.
            cy.request({
                method: 'POST',
                url:  '/prpc/weetbix.v1.Rules/Update',
                body: {
                    rule: {
                        name: 'projects/chromium/rules/ac856b1827dc1cb845486edbf4b80cfa',
                        ruleDefinition: 'test = "cypress test 1"',
                        isActive: true,
                    },
                    updateMask: 'ruleDefinition,isActive'
                },
                headers: {
                    Authorization: 'Bearer ' + accessToken,
                },
            });
        });
        cy.visit('/projects/chromium/clusters/rules-v1/ac856b1827dc1cb845486edbf4b80cfa');
    })
    it('loads rule', () => {
        cy.get('rule-section').get('[data-cy=rule-definition]').contains('test = "cypress test 1"')
        cy.get('rule-section').get('[data-cy=rule-enabled]').contains('Yes')
    })
    it('edit rule definition', () => {
        cy.get('rule-section').get('[data-cy=rule-definition-edit]').click()
        cy.get('rule-section').get('[data-cy=rule-definition-textbox]').get('textarea').type('{selectall}test = "cypress test 2"')
        cy.get('rule-section').get('[data-cy=rule-definition-save]').click()
        cy.get('rule-section').get('[data-cy=rule-definition]').contains('test = "cypress test 2"');
        cy.get('reclustering-progress-indicator').get('[data-cy=reclustering-progress-description]').contains('Weetbix is re-clustering test results')
    })
    it('validation error while editing rule definition', () => {
        cy.get('rule-section').get('[data-cy=rule-definition-edit]').click()
        cy.get('rule-section').get('[data-cy=rule-definition-textbox]').get('textarea').type('{selectall}test = "cypress test 2"a')
        cy.get('rule-section').get('[data-cy=rule-definition-save]').click()
        cy.get('rule-section').get('[data-cy=rule-definition-validation-error]').contains('Validation error: rule definition is not valid: syntax error: 1:24: unexpected token "a"')
        cy.get('rule-section').get('[data-cy=rule-definition-cancel]').click()
        cy.get('rule-section').get('[data-cy=rule-definition]').contains('test = "cypress test 1"');
    })
    it('disable and re-enable', () => {
        cy.get('rule-section').get('[data-cy=rule-enabled-toggle]').contains('Disable').click()
        cy.get('rule-section').get('[data-cy=rule-enabled]').contains('No')

        cy.get('rule-section').get('[data-cy=rule-enabled-toggle]').contains('Enable').click()
        cy.get('rule-section').get('[data-cy=rule-enabled]').contains('Yes')
    })
})
