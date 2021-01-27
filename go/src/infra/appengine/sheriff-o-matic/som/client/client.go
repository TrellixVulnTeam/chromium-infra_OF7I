// Copyright 2015 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package client

import (
	"bytes"
	"context"
	"encoding/json"
	"expvar"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	"infra/monitoring/messages"
	"infra/monorail"

	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/gae/service/info"
	"go.chromium.org/luci/grpc/prpc"
	"go.chromium.org/luci/server/auth"
)

const (
	maxRetries = 3
	// FYI https://github.com/golang/go/issues/9405 in Go 1.4
	// http timeout errors are logged as "use of closed network connection"
	timeout    = 5 * time.Second
	miloScheme = "https"
	miloHost   = "ci.chromium.org"
)

var (
	expvars = expvar.NewMap("client")
)

// Writer writes out data to other services, most notably sheriff-o-matic.
type Writer interface {
	// PostAlerts posts alerts to Sheriff-o-Matic.
	PostAlerts(ctx context.Context, alerts *messages.AlertsSummary) error
}

type trackingHTTPClient struct {
	c *http.Client
	// Some counters for reporting. Only modify through synchronized methods.
	currReqs int64
}

type writer struct {
	hc         *trackingHTTPClient
	alertsBase string
}

// Test represents information about Tests in a builder group.
type Test struct {
	Builders []string `json:"builders"`
}

// CrBugs is a minimal Monorail client for fetching issues.
type CrBugs struct {
	hc *trackingHTTPClient
}

// CrbugItems returns a slice of issues that match label.
func (cr *CrBugs) CrbugItems(ctx context.Context, label string) ([]messages.CrbugItem, error) {
	v := url.Values{}
	v.Add("can", "open")
	v.Add("maxResults", "100")
	v.Add("q", fmt.Sprintf("label:%s", label))

	URL := "https://www.googleapis.com/projecthosting/v2/projects/chromium/issues?" + v.Encode()
	expvars.Add("CrbugIssues", 1)
	defer expvars.Add("CrbugIssues", -1)
	res := &messages.CrbugSearchResults{}
	if code, err := cr.hc.getJSON(ctx, URL, res); err != nil {
		logging.Errorf(ctx, "Error (%d) fetching %s: %v", code, URL, err)
		return nil, err
	}

	return res.Items, nil
}

// FinditAPIResponseV2 represents a response from the findit api.
type FinditAPIResponseV2 struct {
	Responses []*messages.FinditResultV2 `json:"responses"`
}

// NewWriter returns a new Client, which will post alerts to alertsBase.
func NewWriter(alertsBase string, transport http.RoundTripper) Writer {
	return &writer{hc: &trackingHTTPClient{
		c: &http.Client{
			Transport: transport,
		},
	}, alertsBase: alertsBase}
}

func (w *writer) PostAlerts(ctx context.Context, alerts *messages.AlertsSummary) error {
	return w.hc.trackRequestStats(func() (length int64, err error) {
		logging.Infof(ctx, "POSTing alerts to %s", w.alertsBase)
		expvars.Add("PostAlerts", 1)
		defer expvars.Add("PostAlerts", -1)
		b, err := json.Marshal(alerts)
		if err != nil {
			return
		}

		resp, err := w.hc.c.Post(w.alertsBase, "application/json", bytes.NewBuffer(b))
		if err != nil {
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 400 {
			err = fmt.Errorf("http status %d: %s", resp.StatusCode, w.alertsBase)
			return
		}

		length = resp.ContentLength

		return
	})
}

func (hc *trackingHTTPClient) trackRequestStats(cb func() (int64, error)) error {
	var err error
	expvars.Add("InFlight", 1)
	defer expvars.Add("InFlight", -1)
	defer func() {
		if err != nil {
			expvars.Add("TotalErrors", 1)
		}
	}()
	length := int64(0)
	length, err = cb()
	expvars.Add("TotalBytesRead", length)
	return err
}

func (hc *trackingHTTPClient) attemptJSONGet(ctx context.Context, url string, v interface{}) (bool, int, int64, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logging.Errorf(ctx, "error while creating request: %q, possibly retrying.", err.Error())
		return false, 0, 0, err
	}

	return hc.attemptReq(ctx, req, v)
}

func (hc *trackingHTTPClient) attemptReq(ctx context.Context, r *http.Request, v interface{}) (bool, int, int64, error) {
	resp, err := hc.c.Do(r)
	if err != nil {
		logging.Errorf(ctx, "error: %q, possibly retrying.", err.Error())
		return false, 0, 0, err
	}

	defer resp.Body.Close()
	status := resp.StatusCode
	if status != http.StatusOK {
		return false, status, 0, fmt.Errorf("bad response code: %v", status)
	}

	if err = json.NewDecoder(resp.Body).Decode(v); err != nil {
		logging.Errorf(ctx, "Error decoding response: %v", err)
		return false, status, 0, err
	}
	ct := strings.ToLower(resp.Header.Get("Content-Type"))
	expected := "application/json"
	if !strings.HasPrefix(ct, expected) {
		err = fmt.Errorf("unexpected Content-Type, expected \"%s\", got \"%s\": %s", expected, ct, r.URL)
		return false, status, 0, err
	}

	return true, status, resp.ContentLength, err
}

// getJSON does a simple HTTP GET on a getJSON endpoint.
//
// Returns the status code and the error, if any.
func (hc *trackingHTTPClient) getJSON(ctx context.Context, url string, v interface{}) (status int, err error) {
	err = hc.trackRequestStats(func() (length int64, err error) {
		attempts := 0
		for {
			if attempts > 0 {
				logging.Infof(ctx, "Fetching json (%d in flight, attempt %d of %d): %s", hc.currReqs, attempts, maxRetries, url)
			}
			done, tStatus, length, err := hc.attemptJSONGet(ctx, url, v)
			status = tStatus
			if done {
				return length, err
			}
			if err != nil {
				logging.Errorf(ctx, "Error attempting fetch: %v", err)
			}

			attempts++
			if attempts > maxRetries {
				return 0, fmt.Errorf("error fetching %s, max retries exceeded", url)
			}

			if status >= 400 && status < 500 {
				return 0, fmt.Errorf("HTTP status %d, not retrying: %s", status, url)
			}

			time.Sleep(time.Duration(math.Pow(2, float64(attempts))) * time.Second)
		}
	})

	return status, err
}

// getAsSelfOAuthClient returns a client capable of making HTTP requests authenticated
// with OAuth access token for userinfo.email scope.
func getAsSelfOAuthClient(c context.Context) (*http.Client, error) {
	// Note: "https://www.googleapis.com/auth/userinfo.email" is the default
	// scope used by GetRPCTransport(AsSelf). Use auth.WithScopes(...) option to
	// override.
	c, cancel := context.WithTimeout(c, 10*time.Minute)
	defer cancel()

	t, err := auth.GetRPCTransport(c, auth.AsSelf)
	if err != nil {
		return nil, err
	}
	return &http.Client{Transport: t}, nil
}

// NewMonorail registers a new Monorail client instance pointed at baseURL.
func NewMonorail(c context.Context, baseURL string) monorail.MonorailClient {
	client, err := getAsSelfOAuthClient(c)

	if err != nil {
		panic("No OAuth client in context")
	}

	mr := monorail.NewEndpointsClient(client, baseURL+"/_ah/api/monorail/v1/")

	return mr
}

// NewMonorailV3ClientByHost creates a Monorail V3 prpc client given host
// host is something like api-dot-monorail-staging.appspot.com
// audience is something like https://monorail-staging.appspot.com
func NewMonorailV3ClientByHost(c context.Context, host string, audience string) (*prpc.Client, error) {
	c, cancel := context.WithTimeout(c, 5*time.Minute)
	defer cancel()
	t, err := auth.GetRPCTransport(c, auth.AsSelf, auth.WithIDTokenAudience(audience))
	if err != nil {
		return nil, err
	}
	// httpClient is able to make HTTP requests authenticated with
	// ID tokens.
	httpClient := &http.Client{Transport: t}
	return &prpc.Client{
		C:    httpClient,
		Host: host,
	}, nil
}

// NewMonorailV3Client creates a Monorail V3 prpc client
func NewMonorailV3Client(c context.Context) (*prpc.Client, error) {
	if info.AppID(c) == "sheriff-o-matic" {
		return NewMonorailV3ClientByHost(c, "api-dot-monorail-prod.appspot.com", "https://monorail-prod.appspot.com")
	}
	return NewMonorailV3ClientByHost(c, "api-dot-monorail-staging.appspot.com", "https://monorail-staging.appspot.com")
}

// ProdClients returns a set of service clients pointed at production.
func ProdClients(ctx context.Context) (FindIt, CrBug, monorail.MonorailClient) {
	findIt := NewFindit("https://findit-for-me.appspot.com")
	monorailClient := NewMonorail(ctx, "https://monorail-prod.appspot.com")
	crBugs := &CrBugs{}

	return findIt, crBugs, monorailClient
}

// StagingClients returns a set of service clients pointed at instances suitable for a
// staging environment.
func StagingClients(ctx context.Context) (FindIt, CrBug, monorail.MonorailClient) {
	findIt := NewFindit("https://findit-for-me-staging.appspot.com")
	monorailClient := NewMonorail(ctx, "https://monorail-staging.appspot.com")
	crBugs := &CrBugs{}

	return findIt, crBugs, monorailClient
}
