// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import './bug_page';
import '../../../../web_component_types';

import React, { useCallback } from 'react';
import {
    useNavigate,
    useParams
} from 'react-router-dom';

const BugPageWrapper = () => {
    const { bugTracker, id } = useParams();
    const navigate = useNavigate();
    const elementRef = useCallback(node => {
        if (node !== null) {
            node.navigate = navigate;
        }
    }, []);
    return (
        <bug-page
            bugTracker={bugTracker}
            bugId={id}
            ref={elementRef}
        />
    );
};

export default BugPageWrapper;