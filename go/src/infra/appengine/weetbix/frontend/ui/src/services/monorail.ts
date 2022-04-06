// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import { AuthorizedPrpcClient } from '../clients/authorized_client';

declare global {
    interface Window { monorailHostname: string | undefined; }
}

export function getIssuesService() : IssuesService {
    const useIDToken = true;
    if (!window.monorailHostname)
        throw new Error('monorail hostname not set');
    const client = new AuthorizedPrpcClient('api-dot-' + window.monorailHostname, useIDToken);
    return new IssuesService(client);
}

/**
 * Provides access to the monorail issues service over pRPC.
 * For handling errors, import:
 * import { GrpcError } from '@chopsui/prpc-client';
 */
export class IssuesService {
    private static SERVICE = 'monorail.v3.Issues';

    client: AuthorizedPrpcClient;

    constructor(client: AuthorizedPrpcClient) {
        this.client = client;
    }

    async getIssue(request: GetIssueRequest) : Promise<Issue> {
        return this.client.call(IssuesService.SERVICE, 'GetIssue', request, {});
    }
}

export interface GetIssueRequest {
    // The name of the issue to request.
    // Format: projects/{project}/issues/{issue_id}.
    name: string;
}

export interface StatusValue {
    status: string;
    derivation: string;
}

// Definition here is partial. Full definition here:
// https://source.chromium.org/chromium/infra/infra/+/main:appengine/monorail/api/v3/api_proto/issue_objects.proto
export interface Issue {
    name: string;
    summary: string;
    status: StatusValue;
    reporter: string;
    modifyTime: string; // RFC 3339 encoded date/time.
}
