// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

export function setupTestRule() {
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
                    bug: {
                        system: 'monorail',
                        id: 'chromium/123',
                    },
                    isActive: true,
                },
                updateMask: 'ruleDefinition,bug,isActive'
            },
            headers: {
                Authorization: 'Bearer ' + accessToken,
            },
        });
    });
}