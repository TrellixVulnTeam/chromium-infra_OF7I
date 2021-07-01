package cron

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (req *TriggerCronJobReq) Validate() error {
	if req.JobName == "" {
		return status.Errorf(codes.InvalidArgument, "Need cron job name to trigger")
	}
	return nil
}
