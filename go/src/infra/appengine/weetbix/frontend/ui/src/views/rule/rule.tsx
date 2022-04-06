// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React from 'react';
import { useParams } from 'react-router-dom';

import Container from '@mui/material/Container';
import Grid from '@mui/material/Grid';

import ImpactSection from '../../components/impact_section/impact_section';
import RuleTopPanel from '../../components/rule/rule_top_panel/rule_top_panel';

const Rule = () => {

    const { project, id } = useParams();

    return (
        <Container className='mt-1' maxWidth={false}>
            <Grid container spacing={2}>
                <Grid item xs={12}>
                    {(project && id) && (
                        <RuleTopPanel project={project} ruleId={id} />
                    )}
                </Grid>
                <Grid item xs={12}>
                    <ImpactSection />
                </Grid>
            </Grid>
        </Container>
    );
};

export default Rule;