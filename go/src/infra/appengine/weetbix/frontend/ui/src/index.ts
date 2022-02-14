// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import '../styles/style.css';
import './views/bug/bug_page/bug_page.ts';
import './views/bug/bug_table/bugs_table';
import './views/cluster/cluster_page/cluster_page.ts';
import './views/cluster/cluster_table/cluster_table.ts';
import './views/new_rule/new_rule_page.ts';
import './views/errors/not_found_page.ts';
import './shared_elements/title_bar.ts';

import {
    Context,
    Router
} from '@vaadin/router';

const outlet = document.getElementById('outlet');
const router = new Router(outlet);
// serverRoute can be used as the action of routes that should be handled by the server
// instead of in the client.
const serverRoute = (ctx: Context) => { window.location.pathname = ctx.pathname; }
router.setRoutes([
    { path: '/auth/(.*)', action: serverRoute },  // For logout links.
    { path: '/', component: 'cluster-table' },
    { path: '/b/:bugTracker/:id', component: 'bug-page' },
    { path: '/p/:project/rules/new', component: 'new-rule-page' },
    { path: '/p/:project/rules/:id', component: 'cluster-page' },
    { path: '/p/:project/clusters/:algorithm/:id', component: 'cluster-page' },
    { path: '/bugs', component: 'bugs-table' },
    { path: '(.*)', component: 'not-found-page' },
]);