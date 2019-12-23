// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package hwid

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"go.chromium.org/luci/common/errors"
)

var (
	hwidServerURL = "https://chromeos-hwid.appspot.com/api/chromeoshwid/v1/%s/%s/?key=%s"
)

// Data we interested from HWID server.
type Data struct {
	// The Sku string returned by hwid server. It's not the SKU (aka variant).
	Sku string
	// The variant string returned by hwid server. It's not the variant (aka
	// SKU).
	Variant string
}

func callHwidServer(rpc string, hwid string, secret string) ([]byte, error) {
	url := fmt.Sprintf(hwidServerURL, rpc, url.PathEscape(hwid), secret)
	rsp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()

	body, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return nil, err
	}
	if rsp.StatusCode != http.StatusOK {
		return nil, errors.Reason("HWID server responsonse was not OK: %s", body).Err()
	}
	return body, nil
}

// GetHwidData gets the hwid data from hwid server.
func GetHwidData(ctx context.Context, hwid string, secret string) (*Data, error) {
	// TODO (guocb) cache the hwid data.
	data := Data{}
	rspBody, err := callHwidServer("dutlabel", hwid, secret)
	if err != nil {
		return nil, err
	}
	var dutlabels map[string][]interface{}
	if err := json.Unmarshal(rspBody, &dutlabels); err != nil {
		return nil, err
	}
	for key, value := range dutlabels {
		if key != "labels" {
			continue
		}
		for _, labelData := range value {
			label := labelData.(map[string]interface{})
			switch label["name"].(string) {
			case "sku":
				data.Sku = label["value"].(string)
			case "variant":
				data.Variant = label["value"].(string)
			}
		}
	}

	return &data, nil
}
