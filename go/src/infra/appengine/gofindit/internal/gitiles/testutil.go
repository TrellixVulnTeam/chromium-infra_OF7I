// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package gitiles

import (
	"context"
)

type MockedGitilesClient struct{}

func (cl *MockedGitilesClient) sendRequest(c context.Context, url string, params map[string]string) (string, error) {
	// TODO: properly mock some value when we need it
	return `{"logs":[]}`, nil
}

func MockedGitilesClientContext(c context.Context) context.Context {
	return context.WithValue(c, MockedGitilesClientKey, &MockedGitilesClient{})
}
