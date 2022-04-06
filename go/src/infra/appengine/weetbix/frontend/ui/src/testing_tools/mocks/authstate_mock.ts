// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import fetchMock from 'fetch-mock-jest';

export const createMockAuthState = () => {
    return {
        'identity': 'user:user@example.com',
        'email': 'user@example.com',
        'picture': '',
        'accessToken': 'token_text_access',
        'accessTokenExpiry': 1648105586,
        'idToken': 'token_text',
        'idTokenExpiry': 1648105586
    };
};

export const mockFetchAuthState = () => {
    fetchMock.get('/api/authState', createMockAuthState());
};