// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package logdog

import (
	"context"
	"fmt"
)

type MockedLogdogClient struct {
	Data map[string]string // Data for MockedLogdogClient to return
}

func (cl *MockedLogdogClient) GetLog(c context.Context, viewUrl string) (string, error) {
	if val, ok := cl.Data[viewUrl]; ok {
		return val, nil
	}
	return "", fmt.Errorf("Could not get log %s", viewUrl)
}

func (cl *MockedLogdogClient) SetData(data map[string]string) {
	cl.Data = data
}

func MockClientContext(c context.Context, data map[string]string) context.Context {
	return context.WithValue(c, MockedLogdogClientKey, &MockedLogdogClient{
		Data: data,
	})
}
