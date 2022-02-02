// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import { PrpcClient } from '@chopsui/prpc-client';
import { obtainAuthState } from '../libs/auth_state';


export class AuthorizedPrpcClient {
    client: PrpcClient;

    // Initialises a new AuthorizedPrpcClient that connects to host.
    // To connect to Weetbix, leave host unspecified.
    constructor(host?: string) {
        // Only allow insecure connections in Weetbix in local development,
        // where risk of man-in-the-middle attack to server is negligible.
        const insecure = document.location.protocol === "http:" && !host;
        if (insecure && document.location.hostname !== "localhost") {
            // Server misconfiguration.
            throw new Error("Weetbix should never be served over http: outside local development.");
        }
        this.client = new PrpcClient({
            host: host,
            insecure: insecure
        });
    }

    async call(service: string, method: string, message: object, additionalHeaders?: {
        [key: string]: string;
    } | undefined): Promise<any> {
        // Although PrpcClient allows us to pass a token to the constructor,
        // we prefer to inject it at request time to ensure the most recent
        // token is used.
        const authState = await obtainAuthState();
        additionalHeaders = {
            Authorization: 'Bearer ' + authState.accessToken,
            ...additionalHeaders,
        };
        return this.client.call(service, method, message, additionalHeaders);
    }
}
