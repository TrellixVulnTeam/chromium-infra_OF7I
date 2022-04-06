// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import { AuthorizedPrpcClient } from '../clients/authorized_client';

export const getRulesService = () : RulesService => {
    const client = new AuthorizedPrpcClient();
    return new RulesService(client);
};

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

    async lookupBug(request: LookupBugRequest): Promise<LookupBugResponse> {
        return this.client.call(RulesService.SERVICE, 'LookupBug', request, {});
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
    isManagingBug: boolean;
    sourceCluster: ClusterId;
    createTime: string; // RFC 3339 encoded date/time.
    createUser: string;
    lastUpdateTime: string; // RFC 3339 encoded date/time.
    lastUpdateUser: string;
    predicateLastUpdateTime: string; // RFC 3339 encoded date/time.
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
    isManagingBug?: boolean;
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
    isManagingBug?: boolean;
    sourceCluster?: ClusterId;
}

export interface LookupBugRequest {
    system: string;
    id: string;
}

export interface LookupBugResponse {
    // The looked up rules.
    // Format: projects/{project}/rules/{rule_id}.
    rules?: string[];
}

const ruleNameRE = /^projects\/(.*)\/rules\/(.*)$/;

// RuleKey represents the key parts of a rule resource name.
export interface RuleKey {
    project: string;
    ruleId: string;
}

// parseRuleName parses a rule resource name into its key parts.
export const parseRuleName = (name: string):RuleKey => {
    const results = name.match(ruleNameRE);
    if (results == null) {
        throw new Error('invalid rule resource name: ' + name);
    }
    return {
        project: results[1],
        ruleId: results[2],
    };
};
