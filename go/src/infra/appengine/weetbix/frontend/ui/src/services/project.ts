// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import { AuthorizedPrpcClient } from '../clients/authorized_client';

export const getProjectsService = () => {
    const client = new AuthorizedPrpcClient();
    return new ProjectService(client);
};

// A service to handle projects related gRPC requests.
export class ProjectService {
    private static SERVICE = 'weetbix.v1.Projects';

    client: AuthorizedPrpcClient;

    constructor(client: AuthorizedPrpcClient) {
        this.client = client;
    }

    async list(request: ListProjectsRequest): Promise<ListProjectsResponse> {
        return this.client.call(ProjectService.SERVICE, 'List', request);
    }
}

// eslint-disable-next-line @typescript-eslint/no-empty-interface
export interface ListProjectsRequest {}

export interface Project {
    // The format is: `projects/{project}`.
    name: string;
    displayName: string;
    project: string,
}

export interface ListProjectsResponse {
    projects: Project[] | null
}