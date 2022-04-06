// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import './codeblock.css';

import React from 'react';

interface Props {
    code: string | undefined;
}

const CodeBlock = ({ code }: Props) => {
    return (
        <pre className="codeblock" data-testid="codeblock">
            {code}
        </pre>
    );
};

export default CodeBlock;