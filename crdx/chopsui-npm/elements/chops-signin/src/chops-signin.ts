// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import { LitElement, html, svg, css } from 'lit-element';

export interface AuthorizationHeader {
  Authorization?: string;
}

/**
 * `chops-signin` is a web component that manages signing into services using
 * client-side OAuth via gapi.auth2. chops-signin visually indicates whether the
 * user is signed in using either an icon or the user's profile picture. The
 * signin or signout flow is initiated when the user clicks on this component.
 * This component does not require Polymer, but if you are using Polymer, see
 * chops-signin-aware.
 *
 * Usage:
 *  html:
 *   <chops-signin client-id=""...""></chops-signin>
 *  js:
 *   import * as signin from '@chopsui/chops-signin';
 *   window.addEventListener('user-update', ...);
 *   const headers = await signin.getAuthorizationHeaders();
 */
export class ChopsSignin extends LitElement {
  errorMsg?: string;
  clientId = '';
  profile?: gapi.auth2.BasicProfile;

  constructor() {
    super();
    this._onUserUpdate = this._onUserUpdate.bind(this);
  }

  connectedCallback() {
    super.connectedCallback();
    this.addEventListener('click', this._onClick.bind(this));
    window.addEventListener('user-update', this._onUserUpdate);
    this.clientIdChanged();
  }

  disconnectedCallback() {
    window.removeEventListener('user-update', this._onUserUpdate);
  }

  static get styles() {
    return css`
      :host {
        --chops-signin-size: 32px;
        fill: var(--chops-signin-fill-color, red);
        cursor: pointer;
        stroke-width: 0;
        width: var(--chops-signin-size);
        height: var(--chops-signin-size);
      }
      img {
        height: var(--chops-signin-size);
        border-radius: 50%;
        overflow: hidden;
      }
      svg {
        width: var(--chops-signin-size);
        height: var(--chops-signin-size);
      }
    `;
  }

  render() {
    const profile = getUserProfileSync();
    return html`
      ${this.errorMsg
        ? html`
            <div class="error">Error: ${this.errorMsg}</div>
          `
        : html`
            ${!profile
              ? this._icon
              : profile.getImageUrl()
              ? html`
                  <img
                    title="Sign out of ${profile.getEmail()}"
                    src="${profile.getImageUrl()}"
                  />
                `
              : this._icon}
          `}
    `;
  }

  static get properties() {
    return {
      profile: {
        type: Object,
      },
      clientId: {
        attribute: 'client-id',
        type: String,
      },
    };
  }

  clientIdChanged() {
    if (this.clientId) {
      delete this.errorMsg;
      init(this.clientId);
    } else {
      this.errorMsg = 'No client-id attribute set';
    }
  }

  attributeChangedCallback(
    name: string,
    oldval: string | null,
    newval: string | null
  ) {
    super.attributeChangedCallback(name, oldval, newval);
    if (name === 'client-id') {
      this.clientIdChanged();
    }
  }

  get _icon() {
    return svg`<svg viewBox="0 0 24 24">
      <path
        d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm0
        3c1.66 0 3 1.34 3 3s-1.34 3-3 3-3-1.34-3-3 1.34-3 3-3zm0 14.2c-2.5
        0-4.71-1.28-6-3.22.03-1.99 4-3.08 6-3.08 1.99 0 5.97 1.09 6 3.08-1.29
        1.94-3.5 3.22-6 3.22z"></path>
    </svg>`;
  }

  _onUserUpdate() {
    this.profile = getUserProfileSync();
    if (this.profile) {
      this.setAttribute('title', 'Sign out of Google');
    } else {
      this.setAttribute('title', 'Sign in with Google');
    }
  }

  _onClick() {
    return authInitializedPromise
      .then(() => {
        const auth = gapi.auth2.getAuthInstance();
        if (auth.currentUser.get().isSignedIn()) {
          return auth.signOut();
        } else {
          return auth.signIn();
        }
      })
      .catch(err => {
        window.console.error(err);
      });
  }
}

// This check is for unit tests, which fail with '"chops-signin" has
// already been used with this registry` errors.
if (!customElements.get('chops-signin')) {
  customElements.define('chops-signin', ChopsSignin);
}

export function getAuthInstanceAsync(): Promise<gapi.auth2.GoogleAuth> {
  return authInitializedPromise.then(() => {
    return gapi.auth2.getAuthInstance();
  });
}

export function getAuthInstanceSync(): gapi.auth2.GoogleAuth | undefined {
  if (!gapi || !gapi.auth2) return undefined;
  return gapi.auth2.getAuthInstance();
}

export function getAuthorizationHeadersSync(): AuthorizationHeader | undefined {
  const auth = getAuthInstanceSync();
  if (!auth) return undefined;
  const user = auth.currentUser.get();
  if (!user) return {};
  const response = user.getAuthResponse();
  if (!response || !response.access_token) return {};
  return {
    Authorization: response['token_type'] + ' ' + response.access_token,
  };
}

export function getUserProfileSync(): gapi.auth2.BasicProfile | undefined {
  const auth = getAuthInstanceSync();
  if (!auth) return undefined;
  const user = auth.currentUser.get();
  if (!user.isSignedIn()) return undefined;
  return user.getBasicProfile();
}

// This async version waits for gapi.auth2 to finish initializing before
// getting the profile.
export function getUserProfileAsync(): Promise<
  gapi.auth2.BasicProfile | undefined
> {
  return authInitializedPromise.then(getUserProfileSync);
}

const RELOAD_EARLY_MS = 60e3;
let reloadTimerId: number | undefined;

export function getAuthorizationHeaders(): Promise<AuthorizationHeader> {
  return getAuthInstanceAsync()
    .then(auth => {
      if (!auth) return undefined;
      const user = auth.currentUser.get();
      const response = user.getAuthResponse();
      if (response.expires_at === undefined) {
        // The user is not signed in.
        return undefined;
      }
      if (response.expires_at - RELOAD_EARLY_MS < new Date().valueOf()) {
        // The token has expired or is about to expire, so reload it.
        return user.reloadAuthResponse();
      }
      return response;
    })
    .then(response => {
      if (!response) return {};
      if (!reloadTimerId) {
        // Automatically reload when the token is about to expire.
        const delayMs =
          response.expires_at - RELOAD_EARLY_MS + 1 - new Date().valueOf();
        reloadTimerId = window.setTimeout(reloadAuthorizationHeaders, delayMs);
      }
      return {
        Authorization: response['token_type'] + ' ' + response.access_token,
      };
    });
}

export function reloadAuthorizationHeaders() {
  reloadTimerId = undefined;
  getAuthorizationHeaders().then(headers => {
    window.dispatchEvent(
      new CustomEvent('authorization-headers-reloaded', { detail: { headers } })
    );
  });
}

let resolveAuthInitializedPromise: () => void;
export const authInitializedPromise = new Promise<void>(resolve => {
  resolveAuthInitializedPromise = resolve;
});

let gapi: typeof window.gapi;

export function init(
  clientId: string,
  loadLibraries?: string[],
  extraScopes?: string[]
) {
  const callbackName = 'gapi' + Math.random();
  const gapiScript = document.createElement('script');
  gapiScript.src = 'https://apis.google.com/js/api.js?onload=' + callbackName;

  // TODO: see about moving the script element during disconnectedCallback.
  function removeScript() {
    document.head.removeChild(gapiScript);
  }

  window[callbackName] = () => {
    gapi = window.gapi;
    let libraries = 'auth2';
    if (loadLibraries && loadLibraries.length > 0) {
      libraries += ':' + loadLibraries.join(':');
    }
    window.gapi.load(libraries, onAuthLoaded);
    delete window[callbackName];
    removeScript();
  };
  gapiScript.onerror = removeScript;
  document.head.appendChild(gapiScript);

  function onAuthLoaded() {
    if (!window.gapi || !gapi.auth2) return;
    if (!document.body) {
      window.addEventListener('load', onAuthLoaded);
      return;
    }
    let scopes = 'email';
    if (extraScopes && extraScopes.length > 0) {
      scopes += ' ' + extraScopes.join(' ');
    }
    const auth = window.gapi.auth2.init({
      client_id: clientId,
      scope: scopes,
    });
    auth.currentUser.listen(user => {
      window.dispatchEvent(
        new CustomEvent('user-update', { detail: { user } })
      );
      // Start the cycle of setting the reload timer.
      getAuthorizationHeaders();
    });
    auth.then(
      function onFulfilled() {
        resolveAuthInitializedPromise();
      },
      function onRejected(error) {
        window.console.error(error);
      }
    );
  }
}
