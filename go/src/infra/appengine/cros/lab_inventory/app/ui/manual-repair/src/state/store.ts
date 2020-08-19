// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {lazyReducerEnhancer} from 'pwa-helpers/lazy-reducer-enhancer';
import {applyMiddleware, combineReducers, compose, createStore} from 'redux';
import thunk from 'redux-thunk';

import {reducer} from './reducer';

export const store = createStore(
    reducer,
    compose(lazyReducerEnhancer(combineReducers), applyMiddleware(thunk)));
