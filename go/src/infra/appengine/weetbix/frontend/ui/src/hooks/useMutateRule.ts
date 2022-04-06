// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import { useContext } from 'react';
import {
    useMutation,
    useQueryClient
} from 'react-query';

import { SnackbarContext } from '../context/snackbar_context';
import {
    getRulesService,
    UpdateRuleRequest
} from '../services/rules';

type MutationCallback = () => void;

export const useMutateRule = (
    successCallback?: MutationCallback,
    errorCallback?: MutationCallback
) => {
    const ruleService = getRulesService();
    const queryClient = useQueryClient();
    const { setSnack } = useContext(SnackbarContext);

    return useMutation((updateRuleRequest: UpdateRuleRequest) => ruleService.update(updateRuleRequest), {
        onSuccess: (data) => {
            queryClient.setQueryData(['rule', data.ruleId], data);
            setSnack({
                open: true,
                message: 'Rule updated successfully',
                severity: 'success'
            });
            if(successCallback) {
                successCallback();
            }
        },
        onError: (error) => {
            setSnack({
                open: true,
                message: `Failed to mutate rule due to: ${error}`,
                severity: 'error'
            });
            if(errorCallback) {
                errorCallback();
            }
        }
    });
};