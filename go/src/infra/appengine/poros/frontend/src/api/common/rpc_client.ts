// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import { obtainAuthState } from './auth_state';

export interface RpcClient {
  request(service: string, method: string, data: unknown): Promise<string>;
}

type Rpc = (service: string, method: string, data: string) => Promise<string>;

const sendRequest: Rpc = async (
  service: string,
  method: string,
  data: string
) => {
  // the request path looks like "package.names.ServiceName/MethodName",
  // we therefore construct such a string
  const url = `/prpc/${service}/${method}`;
  let additionalHeaders: { [key: string]: string } = {
    'Content-Type': 'application/json',
    Accept: 'application/json',
  };
  if (document.location.hostname !== 'localhost') {
    // Although PrpcClient allows us to pass a token to the constructor,
    // we prefer to inject it at request time to ensure the most recent
    // token is used.
    const authState = await obtainAuthState();
    const token = authState.accessToken;
    additionalHeaders = {
      ...additionalHeaders,
      Authorization: 'Bearer ' + token,
    };
  }

  const requestOptions = {
    method: 'POST',
    headers: additionalHeaders,
    body: JSON.stringify(data),
  };
  return fetch(url, requestOptions)
    .then(async (response) => {
      if (!response.ok) {
        throw new Error(response.statusText);
      }
      const text = await response.text();
      return text.startsWith(")]}'")
        ? JSON.parse(text.substr(4))
        : JSON.parse(text);
    })
    .catch((error: Error) => {
      /*  made up logging service */
      throw error; /* <-- rethrow the error so consumer can still catch it */
    });
};

export const rpcClient: RpcClient = { request: sendRequest };
