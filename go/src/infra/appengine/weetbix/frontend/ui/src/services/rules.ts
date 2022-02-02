// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import { AuthorizedPrpcClient } from './authorized_client';

export function getRulesService() : RulesService {
    let client = new AuthorizedPrpcClient();
    return new RulesService(client);
}

// For handling errors, import:
// import { GrpcError } from '@chopsui/prpc-client';
export class RulesService {
    private static SERVICE = 'weetbix.v1.Rules';

    client: AuthorizedPrpcClient;

    constructor(client: AuthorizedPrpcClient) {
        this.client = client;
    }

    async get(request: GetRuleRequest) : Promise<Rule> {
        return this.client.call(RulesService.SERVICE, 'Get', request, {});
    }

    async list(request: ListRulesRequest): Promise<ListRulesResponse> {
        return this.client.call(RulesService.SERVICE, 'List', request, {});
    }

    async create(request: CreateRuleRequest): Promise<Rule> {
        return this.client.call(RulesService.SERVICE, 'Create', request, {});
    }

    async update(request: UpdateRuleRequest): Promise<Rule> {
        return this.client.call(RulesService.SERVICE, 'Update', request, {});
    }
}

export interface GetRuleRequest {
    // The name of the rule to retrieve.
    // Format: projects/{project}/rules/{rule_id}.
    name: string;
}

export interface Rule {
    name: string;
    project: string;
    ruleId: string;
    ruleDefinition: string;
    bug: AssociatedBug;
    isActive: boolean;
    sourceCluster: ClusterId;
    createTime: string; // RFC 3339 encoded date/time.
    createUser: string;
    lastUpdateTime: string; // RFC 3339 encoded date/time.
    lastUpdateUser: string;
    etag: string;
}

export interface AssociatedBug {
    system: string;
    id: string;
    linkText: string;
    url: string;
}

export interface ClusterId {
    algorithm: string;
    id: string;
}

export interface ListRulesRequest {
    // The parent, which owns this collection of rules.
    // Format: projects/{project}.
    parent: string;
}

export interface ListRulesResponse {
    rules: Rule[];
}

export interface CreateRuleRequest {
    parent: string;
    rule: RuleToCreate;
}

export interface RuleToCreate {
    ruleDefinition: string;
    bug: AssociatedBugToUpdate;
    isActive?: boolean;
    sourceCluster?: ClusterId;
}

export interface AssociatedBugToUpdate {
    system: string;
    id: string;
}

export interface UpdateRuleRequest {
    rule: RuleToUpdate;
    // Comma separated list of fields to be updated.
    // e.g. ruleDefinition,bug,isActive.
    updateMask: string;
    etag?: string;
}

export interface RuleToUpdate {
    name: string;
    ruleDefinition?: string;
    bug?: AssociatedBugToUpdate;
    isActive?: boolean;
    sourceCluster?: ClusterId;
}
