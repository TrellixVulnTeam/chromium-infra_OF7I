# Copyright (c) 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
"""Handles authentication."""

import binascii
import json
import logging
import os
import requests

from google.auth.transport import requests as google_requests
from google.cloud import ndb
from google.oauth2 import id_token

from flask import abort, make_response, redirect, request, session

# pylint: disable=line-too-long
DISCOVERY_ENDPOINT = "https://accounts.google.com/.well-known/openid-configuration"
DEFAULT_AUTHORIZATION_ENDPOINT = "https://accounts.google.com/o/oauth2/v2/auth"
DEFAULT_TOKEN_ENDPOINT = "https://oauth2.googleapis.com/token"

ndb_client = ndb.Client()


class OAuthClient(ndb.Model):  # pylint: disable=W0232
  """Information for OAuth Client"""
  client_id = ndb.StringProperty()
  client_secret = ndb.StringProperty()


class AuthHandler:
  authorization_endpoint = DEFAULT_AUTHORIZATION_ENDPOINT
  token_endpoint = DEFAULT_TOKEN_ENDPOINT

  @classmethod
  def bootstrap(cls):
    try:
      r = requests.get(DISCOVERY_ENDPOINT)
      data = r.json()
      cls.authorization_endpoint = data['authorization_endpoint']
      cls.token_endpoint = data['token_endpoint']
    except Exception as e:
      # If for any reason it failed, use the default endpoint
      logging.error("Got error when querying discovery endpoint: " + e)

    # Get client id and secret from datastore
    with ndb_client.context():
      cls.client = OAuthClient.query().get()
    if not cls.client:
      logging.error('No OAuthClient found in datastore')

  @classmethod
  def get_redirect_uri(cls):
    return 'https://' + request.host + '/code'

  @classmethod
  def get_authorization_url(cls):
    # Creates a state token to prevent request forgery.
    # Stores it in the session for later validation.
    state = binascii.hexlify(os.urandom(32)).decode("utf-8")
    session['state'] = state
    params = {
        "response_type": "code",
        "client_id": cls.client.client_id,
        "scope": "openid email",
        "redirect_uri": cls.get_redirect_uri(),
        "state": state,
    }
    r = requests.Request(
        'GET', cls.authorization_endpoint, params=params).prepare()
    return r.url

  # Handles response from oauth server
  def handle_code(self):
    if not AuthHandler.client:
      logging.error('Cannot get OAuthClient')
      abort(400)
    code = request.args.get('code', '')
    if code == '':
      logging.error("No code found")
      abort(400)

    # Ensures that the request is not a forgery and that the user sending
    # this connect request is the expected user.
    if request.args.get('state', '') != session['state']:
      response = make_response(json.dumps('Invalid state parameter.'), 401)
      response.headers['Content-Type'] = 'application/json'
      return response

    # If there is code, exchange code for access token and ID token
    fields = {
        'code': code,
        'client_id': AuthHandler.client.client_id,
        'client_secret': AuthHandler.client.client_secret,
        'grant_type': 'authorization_code',
        'redirect_uri': AuthHandler.get_redirect_uri(),
    }
    r = requests.post(AuthHandler.token_endpoint, data=fields)
    data = r.json()
    gRequest = google_requests.Request()
    id_info = id_token.verify_oauth2_token(data['id_token'], gRequest)
    email = id_info.get('email')
    logging.info("User email = " + email)
    # session is stored in server and encoded so we can use to store user_email
    session['user_email'] = email
    return redirect('/')
