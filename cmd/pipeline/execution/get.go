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
	"net/http"

	"github.com/spf13/cobra"
	"github.com/spinctl/cmd/gateclient"
	"github.com/spinctl/util"
)

// var (
// 	getExecutionShort = "Get the specified execution"
// 	getExecutionLong  = "Get the execution with the provided id "
// )

type GetOptions struct {
	*executionOptions
	application string
	name        string
}

var (
	getExecutionShort = "Get the latest execution"
	getExecutionLong  = "Get the latest execution "
)

func NewGetCmd(executionOptions executionOptions) *cobra.Command {
	fmt.Println("execution.NewGetCmd")
	options := GetOptions{
		executionOptions: &executionOptions,
	}
	cmd := &cobra.Command{
		Use:   "get",
		Short: getExecutionShort,
		Long:  getExecutionLong,
		RunE: func(cmd *cobra.Command, args []string) error {
			return getLatestExecution(cmd, options)
		},
	}

	cmd.PersistentFlags().StringVarP(&options.application, "application", "a", "", "Spinnaker application the pipeline belongs to")
	cmd.PersistentFlags().StringVarP(&options.name, "name", "n", "", "name of the pipeline")

	return cmd
}

func getLatestExecution(cmd *cobra.Command, options GetOptions) error {
	fmt.Println("getLatestExecution")
	gateClient, err := gateclient.NewGateClient(cmd.InheritedFlags())
	if err != nil {
		return err
	}

	if options.application == "" || options.name == "" {
		return errors.New("one of required parameters 'application' or 'name' not set")
	}

	query := map[string]interface{}{
		"limit":        int32(1),
	}

	successPayload, resp, err := gateClient.ApplicationControllerApi.GetPipelinesUsingGET(
		gateClient.Context, options.application, query)

	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Encountered an error getting the latest execution, status code: %d\n",
			resp.StatusCode)
	}

	for _, p := range successPayload {
		pmap := p.(map[string]interface{})
		if pmap["name"] == options.name {
			fmt.Println(pmap["status"])
			trigger := pmap["trigger"].(map[string]interface{})
			fmt.Println(trigger["tag"])
		}
	}

	// util.UI.JsonOutput(successPayload, util.UI.OutputFormat)
	return nil
}

func getExecution(cmd *cobra.Command, args []string) error {
	gateClient, err := gateclient.NewGateClient(cmd.InheritedFlags())
	if err != nil {
		return err
	}

	id, err := util.ReadArgsOrStdin(args)
	if err != nil {
		return err
	}

	query := map[string]interface{}{
		"executionIds": id, // Status filtering is ignored when executionId is supplied
		"limit":        int32(1),
	}

	successPayload, resp, err := gateClient.ExecutionsControllerApi.GetLatestExecutionsByConfigIdsUsingGET(
		gateClient.Context, query)

	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Encountered an error getting execution %s, status code: %d\n",
			id,
			resp.StatusCode)
	}

	util.UI.JsonOutput(successPayload, util.UI.OutputFormat)
	return nil
}
