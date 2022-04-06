// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import './styles.css';

import dayjs from 'dayjs';
import React from 'react';

import Grid from '@mui/material/Grid';
import Link from '@mui/material/Link';

interface Props {
    createUsername: string | undefined;
    createTime: string | undefined;
    updateUsername: string | undefined;
    updateTime: string | undefined;
}

interface FormattedUsernameProps {
    username: string | undefined;
}

const FormattedUsername = ({ username }: FormattedUsernameProps) => {
    if(!username) {
        return <></>;
    }
    if (username == 'weetbix') {
        return <>Weetbix</>;
    } else if (username.endsWith('@google.com')) {
        const ldap = username.substring(0, username.length - '@google.com'.length);
        return <Link target="_blank" href={`http://who/${ldap}`}>{ldap}</Link>;
    } else {
        return <>{username}</>;
    }
};

const dateFormat = 'LLLL';

const TimestampInfoBar = ({
    createUsername,
    createTime,
    updateUsername,
    updateTime
}: Props) => {
    return (
        <Grid container>
            <Grid item>
                <small
                    title={dayjs.utc(createTime).local().format(dateFormat)}
                    data-testid="timestamp-info-bar-create"
                    className='timestamp_text'
                >
                    Created by {<FormattedUsername username={createUsername} />} {dayjs.utc(createTime).local().fromNow()}. |
                </small>
                <small
                    title={dayjs.utc(updateTime).local().format(dateFormat)}
                    data-testid="timestamp-info-bar-update"
                    className='timestamp_text'
                >
                    {' '}Last modified by {<FormattedUsername username={updateUsername} />} {dayjs.utc(updateTime).local().fromNow()}.
                </small>
            </Grid>
        </Grid>
    );
};

export default TimestampInfoBar;