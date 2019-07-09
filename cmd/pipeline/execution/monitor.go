package execution

import (
	"errors"
	"fmt"
	"time"
	// "github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spinctl/cmd/gateclient"
	"github.com/spinctl/util"
	"net/http"
	// "strings"
)

const (
	STATUS_SUCCEEDED string = "SUCCEEDED"
	STATUS_RUNNING string = "RUNNING"
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

func Monitor(pipelineName string, executionId string, flags *pflag.FlagSet) error {
	if executionId == "" {
		return errors.New("required parameter 'executionId' not set")
	}

	pipelineStatus := ""
	succeededStages := make(map[string]string)
	hasPrinted := false

	for pipelineStatus != STATUS_SUCCEEDED {
		payload, err := getPipelineExecution(executionId, flags)
		pipeline := payload.(map[string]interface{})

		if err != nil {
			return err
		}

		pipelineStatus = pipeline["status"].(string)

		if !hasPrinted && pipelineStatus == STATUS_RUNNING {
			util.UI.Info(util.Colorize().Color(fmt.Sprintf("[reset][bold][green]Pipeline %s is %s", pipelineName, pipelineStatus)))
			hasPrinted = true
		}

		pipelineStages := pipeline["stages"].([]interface{})
		for _, v := range pipelineStages {
			stage := v.(map[string]interface{})
			if _, ok := succeededStages[stage["name"].(string)]; ok {
				continue
			} else if stage["status"].(string) == STATUS_SUCCEEDED {
				succeededStages[stage["name"].(string)] = STATUS_SUCCEEDED
				util.UI.Info(util.Colorize().Color(fmt.Sprintf("[reset][bold][green]Stage %s is %s", stage["name"], stage["status"])))
			} else if stage["status"].(string) == STATUS_RUNNING {
				util.UI.Info(util.Colorize().Color(fmt.Sprintf("[reset][bold][green]Stage %s is %s", stage["name"], stage["status"])))
			}
		}

		if pipelineStatus == STATUS_SUCCEEDED {
			util.UI.Info(util.Colorize().Color(fmt.Sprintf("[reset][bold][blue]Pipeline %s is %s", pipelineName, pipelineStatus)))
			break
		}

		time.Sleep(30 * time.Second)
	}


	return nil
}

func getPipelineExecution(executionId string, flags *pflag.FlagSet) (interface{}, error) {
	gateClient, err := gateclient.NewGateClient(flags)
	if err != nil {
		return nil, err
	}

	retry := 0
	var successPayload interface{}

	for retry <= 3 {
		payload, resp, err := gateClient.PipelineControllerApi.GetPipelineUsingGET(gateClient.Context, executionId)

		if resp.StatusCode != http.StatusOK {
			err = fmt.Errorf("Encountered an error getting pipeline execution id %s, status code: %d\n",
				executionId,
				resp.StatusCode)
		}

		if err == nil {
			successPayload = payload
			break
		}

		retry = retry + 1
		time.Sleep(60 * time.Second)
	}

	return successPayload, err
}
