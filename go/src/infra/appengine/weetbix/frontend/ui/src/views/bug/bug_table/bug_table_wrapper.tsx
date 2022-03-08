// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import './bugs_table';
import '../../../../web_component_types';

import React from 'react';
import {
    useParams
} from 'react-router-dom';

const BugsTableWrapper = () => {
    const { project } = useParams();
    return (
        <bugs-table project={project} />
    );
};

export default BugsTableWrapper;