/* eslint-disable @typescript-eslint/ban-ts-comment */
// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {
    TextDecoder,
    TextEncoder
} from 'util';

import dayjs from 'dayjs';
import localizedFormat from 'dayjs/plugin/localizedFormat';
import relativeTime from 'dayjs/plugin/relativeTime';
import UTC from 'dayjs/plugin/utc';
import fetch from 'node-fetch';

/**
 * jsdom doesn't have those by default, we need to add them for fetch testing.
 */
global.TextEncoder = TextEncoder;

// @ts-ignore
global.TextDecoder = TextDecoder;

// @ts-ignore
global.fetch = fetch;

window.monorailHostname = 'crbug.com';

dayjs.extend(relativeTime);
dayjs.extend(UTC);
dayjs.extend(localizedFormat);