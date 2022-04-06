// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import { PrpcClient } from '@chopsui/prpc-client';
import { obtainAuthState } from '../api/auth_state';

export class AuthorizedPrpcClient {
    client: PrpcClient;
    // Should the ID token be used to authorize the request, or the access token?
    useIDToken: boolean;

    // Initialises a new AuthorizedPrpcClient that connects to host.
    // To connect to Weetbix, leave host unspecified.
    constructor(host?: string, useIDToken?: boolean) {
        // Only allow insecure connections in Weetbix in local development,
        // where risk of man-in-the-middle attack to server is negligible.
        const insecure = document.location.protocol === 'http:' && !host;
        if (insecure && document.location.hostname !== 'localhost') {
            // Server misconfiguration.
            throw new Error('Weetbix should never be served over http: outside local development.');
        }
        this.client = new PrpcClient({
            host: host,
            insecure: insecure
        });
        this.useIDToken = useIDToken === true;
    }

    async call(service: string, method: string, message: object, additionalHeaders?: {
        [key: string]: string;
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    } | undefined): Promise<any> {
        // Although PrpcClient allows us to pass a token to the constructor,
        // we prefer to inject it at request time to ensure the most recent
        // token is used.
        const authState = await obtainAuthState();
        let token: string;
        if (this.useIDToken) {
            token = authState.idToken;
        } else {
            token = authState.accessToken;
        }
        additionalHeaders = {
            Authorization: 'Bearer ' + token,
            ...additionalHeaders,
        };
        return this.client.call(service, method, message, additionalHeaders);
    }
}
