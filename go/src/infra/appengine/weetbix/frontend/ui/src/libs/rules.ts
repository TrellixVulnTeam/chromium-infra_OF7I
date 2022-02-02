// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// RuleCreateRequest is the data expected the server in a POST request
// to create a rule.
export interface RuleCreateRequest {
    rule: RuleToCreate;
    xsrfToken: string;
}

interface RuleToCreate {
    ruleDefinition: string;
    bugId: BugId;
    isActive: boolean;
    sourceCluster: ClusterId;
}

// RuleUpdateRequest is the data expected the server in a PATCH request
// to update a rule.
export interface RuleUpdateRequest {
    rule: RuleToUpdate;
    updateMask: FieldMask;
    xsrfToken: string;
}

interface FieldMask {
    paths: string[];
}

interface RuleToUpdate {
    ruleDefinition?: string;
    bugId?: BugId;
    isActive?: boolean;
}

// Rule is the failure association rule information sent by the server.
export interface Rule {
    project: string;
    ruleId: string;
    ruleDefinition: string;
    creationTime: string; // RFC 3339 encoded date/time.
    creationUser: string;
    lastUpdated: string; // RFC 3339 encoded date/time.
    lastUpdatedUser: string;
    bugId: BugId;
    bugLink: BugLink;
    isActive: boolean;
    sourceCluster: ClusterId;
}

interface BugLink {
    name: string;
    url: string;
}

interface BugId {
    system: string;
    id: string;
}

export interface ClusterId {
    algorithm: string;
    id: string;
}
