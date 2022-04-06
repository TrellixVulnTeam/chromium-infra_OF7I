// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import { useQuery } from 'react-query';
import { getRulesService } from '../services/rules';

const useFetchRule = (ruleId: string | undefined, project: string | undefined) => {

    const rulesService = getRulesService();

    return useQuery(['rule', ruleId], async () => await rulesService.get(
        {
            name: `projects/${project}/rules/${ruleId}`,
        }
    ));
};

export default useFetchRule;