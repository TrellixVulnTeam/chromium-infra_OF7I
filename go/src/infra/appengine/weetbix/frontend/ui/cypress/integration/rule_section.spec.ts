// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
import { setupTestRule } from './test_data';

describe('Rule Section', () => {
    beforeEach(() => {
        // Login.
        cy.visit('/').contains('LOGIN').click();

        setupTestRule();

        cy.visit('/p/chromium/rules/ac856b1827dc1cb845486edbf4b80cfa');
    })

    it('loads rule', () => {
        cy.get('rule-section').get('[data-cy=rule-title]').contains('Weetbix Cypress Test Bug')
        cy.get('rule-section').get('[data-cy=rule-status]').contains('Untriaged')
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

    it('edit bug', () => {
        cy.get('rule-section').get('[data-cy=bug-edit]').click()
        cy.get('rule-section').get('[data-cy=bug-number-textbox]').get('[type=text]').type('{selectall}920869')
        cy.get('rule-section').get('[data-cy=bug-save]').click()
        cy.get('rule-section').get('[data-cy=bug]').contains('crbug.com/920869');
        cy.get('rule-section').get('[data-cy=rule-title]').contains('Weetbix Cypress Alternate Test Bug')
        cy.get('rule-section').get('[data-cy=rule-status]').contains('Fixed')
    })

    it('validation error while editing bug', () => {
        cy.get('rule-section').get('[data-cy=bug-edit]').click()
        cy.get('rule-section').get('[data-cy=bug-number-textbox]').get('[type=text]').type('{selectall}125a')
        cy.get('rule-section').get('[data-cy=bug-save]').click()
        cy.get('rule-section').get('[data-cy=bug-validation-error]').contains('Validation error: not a valid monorail bug ID')
        cy.get('rule-section').get('[data-cy=bug-cancel]').click()
        cy.get('rule-section').get('[data-cy=bug]').contains('crbug.com/920867');
    })

    it('disable and re-enable', () => {
        cy.get('rule-section').get('[data-cy=rule-enabled-toggle]').contains('Disable').click()
        cy.get('rule-section').get('[data-cy=rule-enabled]').contains('No')

        cy.get('rule-section').get('[data-cy=rule-enabled-toggle]').contains('Enable').click()
        cy.get('rule-section').get('[data-cy=rule-enabled]').contains('Yes')
    })
})
