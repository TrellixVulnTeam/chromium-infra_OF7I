// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import './styles/style.css';

import { Context, Router } from '@vaadin/router';
import './element/bugs_table.ts';
import './element/cluster_table.ts';
import './element/cluster_page.ts';
import './element/new_rule_page.ts';
import './element/not_found_page.ts';
import './element/title_bar.ts';


const outlet = document.getElementById('outlet');
const router = new Router(outlet);
// serverRoute can be used as the action of routes that should be handled by the server
// instead of in the client.
const serverRoute = (ctx: Context) => { window.location.pathname = ctx.pathname; }
router.setRoutes([
    { path: '/auth/(.*)', action: serverRoute },  // For logout links.
    { path: '/', component: 'cluster-table' },
    { path: '/projects/:project/rules/new', component: 'new-rule-page' },
    { path: '/projects/:project/clusters/:algorithm/:id', component: 'cluster-page' },
    { path: '/bugs', component: 'bugs-table' },
    { path: '(.*)', component: 'not-found-page' },
]);