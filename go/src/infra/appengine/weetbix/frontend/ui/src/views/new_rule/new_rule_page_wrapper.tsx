// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import './new_rule_page';
import '../../../web_component_types';

import React, { useCallback } from 'react';
import {
    useNavigate,
    useParams,
    useSearchParams
} from 'react-router-dom';

const NewRulePageWrapper = () => {
    const { project } = useParams();
    const navigate = useNavigate();
    const [searchParams] = useSearchParams();

    // This is a way to pass functionality from react to web-components.
    // This strategy, however, does not work if the functionality 
    // is required when the component is initialising.
    const elementRef = useCallback(node => {
        if (node !== null) {
            node.navigate = navigate;
        }
    }, []);
    return (
        <new-rule-page
            project={project}
            ref={elementRef}
            ruleString={searchParams.get('rule')}
            sourceAlg={searchParams.get('sourceAlg')}
            sourceId={searchParams.get('sourceId')}
        />
    );
};

export default NewRulePageWrapper;