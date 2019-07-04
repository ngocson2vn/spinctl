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

type MonitorOptions struct {
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
	monitorExecutionShort = "List the executions for the provided pipeline id"
	monitorExecutionLong  = "List the executions for the provided pipeline id"
)

func NewMonitorCmd(executionOptions executionOptions) *cobra.Command {
	options := MonitorOptions{
		executionOptions: &executionOptions,
	}
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   monitorExecutionShort,
		Long:    monitorExecutionLong,
		RunE: func(cmd *cobra.Command, args []string) error {
			return monitorExecution(cmd, options)
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

func monitorExecution(cmd *cobra.Command, options MonitorOptions) error {
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
