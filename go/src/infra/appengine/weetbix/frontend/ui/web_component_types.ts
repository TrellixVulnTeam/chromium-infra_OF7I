// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import { DOMAttributes } from 'react';

import { TitleBar } from './src/shared_elements/title_bar';
import { BugPage } from './src/views/bug/bug_page/bug_page';
import { BugsTable } from './src/views/bug/bug_table/bugs_table';
import { ClusterPage } from './src/views/cluster/cluster_page/cluster_page';
import { ClusterTable } from './src/views/cluster/cluster_table/cluster_table';
import { HomePage } from './src/views/home/home_page';
import { NewRulePage } from './src/views/new_rule/new_rule_page';

type CustomElement<T> = Partial<T & DOMAttributes<T> & { children: any }>;

declare global {
  namespace JSX {
    interface IntrinsicElements {
      ['home-page']: CustomElement<HomePage>;
      ['title-bar']: CustomElement<TitleBar>;
      ['new-rule-page']: CustomElement<NewRulePage>;
      ['cluster-page']: CustomElement<ClusterPage>;
      ['cluster-table']: CustomElement<ClusterTable>;
      ['bug-page']: CustomElement<BugPage>;
      ['bugs-table']: CustomElement<BugsTable>;
    }
  }
};