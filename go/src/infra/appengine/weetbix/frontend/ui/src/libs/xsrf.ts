// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

let xsrfToken : XSRFTokenCache | null = null;

// obtainXSRFToken obtains a current XSRF token, for interacting
// with Weetbix APIs.
export async function obtainXSRFToken(): Promise<string> {
    if (xsrfToken == null) {
        // Initialise with an expired token.
        xsrfToken = {
            token: "",
            expiry: new Date(0),
        }
    }
    const now = new Date()
    if (now <= xsrfToken.expiry) {
        // Token is still valid.
        return xsrfToken.token;
    }

    // Refresh the token.
    const r = await fetch(`/api/xsrfToken`);
    const response : XSRFToken = await r.json();

    // Let the token expire 3 hours from now. Actual validity
    // is 4 hours, so this leaves some margin.
    const expiry = new Date()
    expiry.setTime(expiry.getTime() + 3*60*60*1000)

    xsrfToken = {
        token: response.token,
        expiry: expiry,
    }

    return response.token
}

// An XSRF token as returned by the server.
interface XSRFToken {
    token: string;
}

interface XSRFTokenCache {
    token: string;
    expiry: Date;
}
