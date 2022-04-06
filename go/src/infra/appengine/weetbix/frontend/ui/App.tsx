// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import './styles/style.css';
import './src/views/home/home_page';
import './src/views/bug/bug_page/bug_page.ts';
import './src/views/bug/bug_table/bugs_table';
import './src/views/clusters/cluster/cluster_page.ts';
import './src/views/clusters/cluster_table/cluster_table.ts';
import './src/views/new_rule/new_rule_page.ts';
import './src/views/clusters/cluster/elements/impact_table';

import React from 'react';
import {
    QueryClient,
    QueryClientProvider
} from 'react-query';
import {
    Route,
    Routes
} from 'react-router-dom';

import BaseLayout from './src/layouts/base';
import BugPageWrapper from './src/views/bug/bug_page/bug_page_wrapper';
import BugsTableWrapper from './src/views/bug/bug_table/bug_table_wrapper';
import ClusterPageWrapper from './src/views/clusters/cluster/cluster_page_wrapper';
import ClusterTableWrapper from './src/views/clusters/cluster_table/cluster_table_wrapper';
import NotFoundPage from './src/views/errors/not_found_page';
import HomePageWrapper from './src/views/home/home_page_wrapper';
import NewRulePageWrapper from './src/views/new_rule/new_rule_page_wrapper';
import Rule from './src/views/rule/rule';
import { SnackbarContextWrapper } from './src/context/snackbar_context';
import FeedbackSnackbar from './src/components/error_snackbar/feedback_snackbar';

const queryClient = new QueryClient(
    {
        defaultOptions: {
            queries: {
                refetchOnWindowFocus: false,
            },
        },
    }
);

const App = () => {
    return (
        <SnackbarContextWrapper>
            <QueryClientProvider client={queryClient}>
                <Routes>
                    <Route path='/' element={<BaseLayout />}>
                        <Route index element={<HomePageWrapper />} />
                        <Route path='b/:bugTracker/:id' element={<BugPageWrapper />} />
                        <Route path='p/:project'>
                            <Route path='rules'>
                                <Route path='new' element={<NewRulePageWrapper />} />
                                <Route path=':id' element={<Rule />} />
                            </Route>
                            <Route path='clusters'>
                                <Route index element={<ClusterTableWrapper />} />
                                <Route path=':algorithm/:id' element={<ClusterPageWrapper />} />
                            </Route>
                            <Route path='bugs' element={<BugsTableWrapper />} />
                        </Route>
                        <Route path='*' element={<NotFoundPage />} />
                    </Route>
                </Routes>
            </QueryClientProvider>
            <FeedbackSnackbar />
        </SnackbarContextWrapper>
    );
};

export default App;