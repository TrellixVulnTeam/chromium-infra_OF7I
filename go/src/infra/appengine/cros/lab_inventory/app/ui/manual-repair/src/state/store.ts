// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {lazyReducerEnhancer} from 'pwa-helpers/lazy-reducer-enhancer';
import {applyMiddleware, combineReducers, compose, createStore, StoreEnhancer} from 'redux';
import thunk from 'redux-thunk';

import {reducer} from './reducer';

// Sets up a Chrome extension for time travel debugging.
// See https://github.com/zalmoxisus/redux-devtools-extension for more
// information.
declare global {
  interface Window {
    process?: {};
    __REDUX_DEVTOOLS_EXTENSION_COMPOSE__?: typeof compose;
  }
}

const devCompose: <Ext0, Ext1, StateExt0, StateExt1>(
    f1: StoreEnhancer<Ext0, StateExt0>, f2: StoreEnhancer<Ext1, StateExt1>) =>
    StoreEnhancer<Ext0&Ext1, StateExt0&StateExt1> =
        window.__REDUX_DEVTOOLS_EXTENSION_COMPOSE__ || compose;

// Initializes the Redux store with a lazyReducerEnhancer (so that we can
// lazily add reducers after the store has been created) and redux-thunk (so
// that we can dispatch async actions).
export const store = createStore(
    reducer,
    devCompose(lazyReducerEnhancer(combineReducers), applyMiddleware(thunk)));
