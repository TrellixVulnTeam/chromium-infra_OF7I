package swarming

import (
	"context"
	"fmt"
	"strings"

	"go.chromium.org/luci/common/errors"

	swarming "go.chromium.org/luci/common/api/swarming/swarming/v1"
)

const (
	// SSWPath is the default path for skylab_swarming_worker
	SSWPath = "/opt/infra-tools/skylab_swarming_worker"

	// DeployTaskExecutionTimeout is the default timeout for deploy tasks.
	DeployTaskExecutionTimeout = 18000
	// DeployTaskExpiryTimeout is the default timeout for task expiry.
	DeployTaskExpiryTimeout = 7200
	// DeployTaskPriority is the default priority for the task.
	DeployTaskPriority = 29
)

// deployTaskCommand constructs the command to be run on the target
func (tc *TaskCreator) deployTaskCommand(actions []string) []string {
	cmd := []string{SSWPath}
	if actions != nil && len(actions) != 0 {
		cmd = append(cmd, "-actions", strings.Join(actions, ","))
	}
	logLocation := tc.LogdogURL()
	if logLocation != "" {
		cmd = append(cmd, "-logdog-annotation-url", logLocation)
	}
	cmd = append(cmd, "-task-name", "deploy")
	return cmd
}

// deployTaskTags appends tags for the task with default ones.
func (tc *TaskCreator) deployTaskTags(dutID string, newTags []string) []string {
	tags := []string{}
	tags = append(tags, newTags...)
	tags = append(tags, "task:Deploy")
	tags = append(tags, fmt.Sprintf("luci_project:%s", LuciProject))
	tags = append(tags, fmt.Sprintf("admin_session:%s", tc.session))
	tags = append(tags, fmt.Sprintf("deploy_task:%s", dutID))
	logLocation := tc.LogdogURL()
	if logLocation != "" {
		tags = append(tags, fmt.Sprintf("log_location:%s", logLocation))
	}
	return tags
}

// deployDUTTask creates a new task RPC request for deploy DUT
func (tc *TaskCreator) deployDUTTask(hostname, dutID, pool, user string, timeout int64, actions, tags []string, dimensions map[string]string) *swarming.SwarmingRpcsNewTaskRequest {
	if dimensions == nil {
		dimensions = make(map[string]string)
	}
	if _, ok := dimensions[DUTIDDimensionKey]; !ok && dutID != "" {
		dimensions[DUTIDDimensionKey] = dutID
	}
	if _, ok := dimensions[DUTNameDimensionKey]; !ok && hostname != "" {
		dimensions[DUTNameDimensionKey] = hostname
	}
	if _, ok := dimensions[PoolDimensionKey]; !ok {
		dimensions[PoolDimensionKey] = pool
	}
	if timeout == 0 {
		timeout = DeployTaskExecutionTimeout
	}

	req := &swarming.SwarmingRpcsNewTaskRequest{
		EvaluateOnly: false,
		Name:         "Deploy",
		Priority:     DeployTaskPriority,
		Tags:         tc.deployTaskTags(dutID, tags),
		TaskSlices: []*swarming.SwarmingRpcsTaskSlice{
			{
				ExpirationSecs: DeployTaskExpiryTimeout,
				Properties: &swarming.SwarmingRpcsTaskProperties{
					Command:              tc.deployTaskCommand(actions),
					Dimensions:           MapToSwarmingDimensions(dimensions),
					ExecutionTimeoutSecs: timeout,
					Idempotent:           false,
				},
				WaitForCapacity: true,
			},
		},
		User:           user,
		ServiceAccount: tc.SwarmingServiceAccount,
	}
	return req
}

// DeployDut creates a task request for deploy task based on the input.
func (tc *TaskCreator) DeployDut(ctx context.Context, hostname, dutID, pool string, timeout int64, actions, tags []string, dimensions map[string]string) (*TaskInfo, error) {
	if tc.authenticator == nil {
		return nil, errors.Reason("Unable to get user email").Err()
	}
	user, err := tc.authenticator.GetEmail()
	if err != nil {
		return nil, errors.Annotate(err, "Unable to get user info").Err()
	}
	tc.GenerateLogdogTaskCode()
	return tc.schedule(ctx, tc.deployDUTTask(hostname, dutID, pool, user, timeout, actions, tags, dimensions))
}
