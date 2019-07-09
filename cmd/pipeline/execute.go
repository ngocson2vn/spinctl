// Copyright (c) 2018, Google, Inc.
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

package pipeline

import (
	// "errors"
	"fmt"
	"net/http"
	"time"
	"strings"
	// "reflect"
	// "encoding/json"

	// "github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spinctl/cmd/gateclient"
	"github.com/spinctl/util"
	"github.com/spinctl/cmd/pipeline/execution"
)

func Execute(applicationName string, pipelineName string, flags *pflag.FlagSet) error {
	util.UI.Info(util.Colorize().Color(fmt.Sprintf("[reset][bold][green]Executing the pipeline: %s", pipelineName)))

	retry := 0
	var successPayload map[string]interface{}
	var err error

	for retry <= 3 {
		successPayload, err = executePipeline(applicationName, pipelineName, flags)

		if err == nil {
			break
		}

		retry = retry + 1
		time.Sleep(60 * time.Second)
	}

	if err != nil {
		return err
	}

	executionId := strings.Split(successPayload["ref"].(string), "/")[2]
	util.UI.Info(util.Colorize().Color(fmt.Sprintf("[reset][bold][green]Pipeline execution id: %s", executionId)))

	time.Sleep(5 * time.Second)
	err = execution.Monitor(pipelineName, executionId, flags)
	if err != nil {
		return err
	}

	return nil
}

func executePipeline(applicationName string, pipelineName string, flags *pflag.FlagSet) (map[string]interface{}, error) {
	gateClient, err := gateclient.NewGateClient(flags)
	if err != nil {
		return make(map[string]interface{}), err
	}

	trigger := map[string]interface{}{"type": "manual"}
	successPayload, resp, err := gateClient.PipelineControllerApi.InvokePipelineConfigUsingPOST1(gateClient.Context,
		applicationName,
		pipelineName,
		map[string]interface{}{"trigger": trigger})

	if err != nil {
		err = fmt.Errorf("Execute pipeline failed with response: %v and error: %s\n", resp, err)
	}

	if resp.StatusCode != http.StatusAccepted {
		err = fmt.Errorf("Encountered an error executing pipeline, status code: %d\n", resp.StatusCode)
	}

	if successPayload == nil {
		err = fmt.Errorf("Could not get success payload!")
	}

	return successPayload, err
}
