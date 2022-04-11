// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import '../shared_elements/title_bar';
import '../../web_component_types';

import React from 'react';
import {
    Outlet,
    useParams
} from 'react-router-dom';

declare global {
    interface Window {
        email: string;
        logoutUrl: string;
    }
}

const BaseLayout = () => {

    const params = useParams();
    return (
        <>
            <title-bar email={window.email} logoutUrl={window.logoutUrl} project={params.project === null ? undefined : params.project}/>
            <Outlet />
        </>
    );

};

export default BaseLayout;
