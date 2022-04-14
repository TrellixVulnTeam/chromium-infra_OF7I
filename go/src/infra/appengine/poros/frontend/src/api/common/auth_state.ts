// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

let authState: AuthState | null = null;

/**
 * obtainAuthState obtains a current auth state, for interacting
 * with pRPC APIs.
 */
export async function obtainAuthState(): Promise<AuthState> {
  if (
    authState != null &&
    authState.accessTokenExpiry * 1000 > Date.now() + 5000 &&
    authState.idTokenExpiry * 1000 > Date.now() + 5000
  ) {
    // Auth state is still has >=5 seconds of validity for
    // both tokens.
    return authState;
  }

  // Refresh the auth state.
  const response = await queryAuthState();
  authState = response;
  return authState;
}

export interface AuthState {
  identity: string;
  email: string;
  picture: string;
  accessToken: string;
  idToken: string;
  // Expiration time (unix timestamp) of the access token.
  // If zero/undefined, the access token does not expire.
  accessTokenExpiry: number;
  idTokenExpiry: number;
}

export async function queryAuthState(): Promise<AuthState> {
  const res = await fetch('/api/authState');
  if (!res.ok) {
    throw new Error('failed to get authState:\n' + (await res.text()));
  }
  return res.json();
}
