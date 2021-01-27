package client

import (
	"bytes"
	"context"
	"encoding/json"

	"infra/monitoring/messages"

	"go.chromium.org/luci/common/logging"
)

type findit struct {
	simpleClient
}

// FinditBuildbucket fetches items from the findit service using buildbucket concept, which identifies possible culprit CLs for a failed build.
func (f *findit) FinditBuildbucket(ctx context.Context, buildID int64, failedSteps []string) ([]*messages.FinditResultV2, error) {
	data := map[string]interface{}{
		"requests": []map[string]interface{}{
			{
				"build_id":     buildID,
				"failed_steps": failedSteps,
			},
		},
	}

	b := bytes.NewBuffer(nil)
	err := json.NewEncoder(b).Encode(data)

	if err != nil {
		return nil, err
	}

	URL := f.Host + "/_ah/api/findit/v1/lucibuildfailure"
	res := &FinditAPIResponseV2{}
	if code, err := f.postJSON(ctx, URL, b.Bytes(), res); err != nil {
		logging.Errorf(ctx, "Error (%d) fetching %s: %v", code, URL, err)
		return nil, err
	}

	return res.Responses, nil
}

// NewFindit registers a findit client pointed at host.
func NewFindit(host string) FindIt {
	return &findit{simpleClient{Host: host, Client: nil}}
}
