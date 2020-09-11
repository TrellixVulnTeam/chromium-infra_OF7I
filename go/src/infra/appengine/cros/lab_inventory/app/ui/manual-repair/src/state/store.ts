// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {Action} from '@reduxjs/toolkit';
import {lazyReducerEnhancer} from 'pwa-helpers/lazy-reducer-enhancer';
import {applyMiddleware, combineReducers, compose, createStore, StoreEnhancer} from 'redux';
import thunk, {ThunkAction, ThunkDispatch} from 'redux-thunk';

import {ApplicationState, reducers} from './reducers/index';

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
    reducers,
    devCompose(lazyReducerEnhancer(combineReducers), applyMiddleware(thunk)));

export type AppDispatch = typeof store.dispatch;
export type AppThunkDispatch = ThunkDispatch<ApplicationState, void, Action>;
export type AppThunk<ReturnType = void> =
    ThunkAction<ReturnType, ApplicationState, unknown, Action<string>>;

// TODO: This is a possibly not a good fix to as we cast store.dispatch as Thunk
// dispatch. Will investigate.
export const thunkDispatch = store.dispatch as AppThunkDispatch;
