// Copyright (c) 2019, Google, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.

package execution

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spinctl/cmd/gateclient"
	"github.com/spinctl/util"
	"net/http"
	"strings"
)

type ListOptions struct {
	*executionOptions
	output           string
	pipelineConfigId string
	limit            int32
	running          bool
	succeeded        bool
	failed           bool
	canceled         bool
}

var (
	listExecutionShort = "List the executions for the provided pipeline id"
	listExecutionLong  = "List the executions for the provided pipeline id"
)

func NewListCmd(executionOptions executionOptions) *cobra.Command {
	options := ListOptions{
		executionOptions: &executionOptions,
	}
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   listExecutionShort,
		Long:    listExecutionLong,
		RunE: func(cmd *cobra.Command, args []string) error {
			return listExecution(cmd, options)
		},
	}

	cmd.PersistentFlags().StringVarP(&options.pipelineConfigId, "pipeline-id", "i", "", "Spinnaker pipeline id to list executions for")
	cmd.PersistentFlags().Int32VarP(&options.limit, "limit", "l", -1, "number of executions to return")
	cmd.PersistentFlags().BoolVar(&options.running, "running", false, "add filter for running executions")
	cmd.PersistentFlags().BoolVar(&options.succeeded, "succeeded", false, "add filter for succeeded executions")
	cmd.PersistentFlags().BoolVar(&options.failed, "failed", false, "add filter for failed executions")
	cmd.PersistentFlags().BoolVar(&options.canceled, "canceled", false, "add filter for canceled executions")

	return cmd
}

func listExecution(cmd *cobra.Command, options ListOptions) error {
	gateClient, err := gateclient.NewGateClient(cmd.InheritedFlags())
	if err != nil {
		return err
	}

	if options.pipelineConfigId == "" {
		return errors.New("required parameter 'pipeline-id' not set")
	}

	query := map[string]interface{}{
		"pipelineConfigIds": options.pipelineConfigId,
	}

	var statuses []string
	if options.running {
		statuses = append(statuses, "RUNNING")
	}
	if options.succeeded {
		statuses = append(statuses, "SUCCEEDED", "STOPPED", "SKIPPED")
	}
	if options.failed {
		statuses = append(statuses, "TERMINAL", "STOPPED", "FAILED_CONTINUE")
	}
	if options.canceled {
		statuses = append(statuses, "CANCELED")
	}
	if len(statuses) > 0 {
		query["statuses"] = strings.Join(statuses, ",")
	}

	if options.limit > 0 {
		query["limit"] = options.limit
	}

	successPayload, resp, err := gateClient.ExecutionsControllerApi.GetLatestExecutionsByConfigIdsUsingGET(
		gateClient.Context, query)

	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Encountered an error listing executions for pipeline id %s, status code: %d\n",
			options.pipelineConfigId,
			resp.StatusCode)
	}

	util.UI.JsonOutput(successPayload, util.UI.OutputFormat)
	return nil
}
