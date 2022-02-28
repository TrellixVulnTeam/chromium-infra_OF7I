// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

describe("Home page", () => {

    beforeEach(() => {
        // Login.
        cy.visit('/').contains('LOGIN').click();
    });

    it("Loads the project list", () => {
        cy.get("h1")
            .should("contain", "Projects");
        cy.get("project-card")
            .then((cardElement) => {
                expect(cardElement).to.exist
            });
    });
})