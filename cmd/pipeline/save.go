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
	"fmt"
	"net/http"
	"strings"

	// "github.com/spf13/cobra"
	"github.com/spinctl/cmd/gateclient"
	"github.com/spf13/pflag"
	"github.com/spinctl/util"
)

func SaveDryRun(pipelineYamls []interface{}, appAttributes map[string]string, dockerImages []string, flags *pflag.FlagSet) error {
	pipelineJsons, err := buildPipelineJsons(pipelineYamls, appAttributes, dockerImages)
	if err != nil {
		return err
	}

	util.UI.JsonOutput(pipelineJsons, util.UI.OutputFormat)

	return nil
}

func Save(pipelineYamls []interface{}, appAttributes map[string]string, dockerImages []string, flags *pflag.FlagSet) error {
	pipelineJsons, err := buildPipelineJsons(pipelineYamls, appAttributes, dockerImages)
	if err != nil {
		return err
	}

	for _, pipeline := range pipelineJsons {
		err = savePipeline(pipeline, flags)
		if err != nil {
			return err
		}
	}

	return nil
}

func buildPipelineJsons(pipelineYamls []interface{}, 
	appAttributes map[string]string, dockerImages []string) ([]map[string]interface{}, error) {

	var pipelineJsons []map[string]interface{}
	pipelineUuidMap := make(map[string]string)

	for _, py := range pipelineYamls {
		pj, err := buildPipelineJsonFromPipelineYaml(py.(map[interface{}]interface{}), appAttributes, dockerImages)
		if err != nil {
			return pipelineJsons, err
		}

		pipelineUuidMap[pj["name"].(string)] = pj["id"].(string)
		pipelineJsons = append(pipelineJsons, pj)
	}

	for _, pipeline := range pipelineJsons {
		for _, stage := range pipeline["stages"].([]map[string]interface{}) {
			if stage["type"] == "pipeline" {
				if value, ok := pipelineUuidMap[stage["pipeline"].(string)]; ok {
					stage["pipeline"] = value
				}
			}
		}
	}

	return pipelineJsons, nil
}

func buildPipelineJsonFromPipelineYaml(pipelineYaml map[interface{}]interface{}, 
	appAttributes map[string]string, dockerImages []string) (map[string]interface{}, error) {

	spinnakerDir := appAttributes["spinnakerDir"]
	pipelineJson, err := util.ParseJsonFromFile(fmt.Sprintf("%s/template/pipeline.json", spinnakerDir), true)
	if err != nil {
		return make(map[string]interface{}), err
	}

	pipelineJson["name"] = fmt.Sprintf("%s-%s", pipelineYaml["name"], appAttributes["pipelineVersion"])
	pipelineUuid, err := util.GenerateUpperCaseUuid(pipelineJson["name"].(string))
	pipelineJson["id"] = pipelineUuid
	pipelineJson["lastModifiedBy"] = appAttributes["ownerEmail"]
	pipelineJson["updateTs"] = appAttributes["timestamp"]

	var stages []map[string]interface{}
	for i, s := range pipelineYaml["stages"].([]interface{}) {
		stageYaml := s.(map[interface{}]interface{})
		parentStage := stageYaml["inherit"]
		stageJson, err := util.ParseJsonFromFile(fmt.Sprintf("%s/template/stages/%s.json", spinnakerDir, parentStage), true)
		if err != nil {
			return make(map[string]interface{}), err
		}

		stageJson["name"] = stageYaml["name"]

		if _, ok := stageJson["notifications"]; ok {
			notifications := stageJson["notifications"].([]interface{})
			if notifications[0].(map[string]interface{})["type"] == "slack" {
				notifications[0].(map[string]interface{})["address"] = appAttributes["slackChannel"]
			}
		}

		stageJson["refId"] = fmt.Sprintf("%d", i + 1)
		if i > 0 {
			stageJson["requisiteStageRefIds"] = []string{ fmt.Sprintf("%d", i) }
		}

		if parentStage == "application" || parentStage == "worker" {
			manifest := stageJson["manifests"].([]interface{})[0].(map[string]interface{})
			if value, ok := stageYaml["metadata"]; ok {
				metadata := value.(map[interface{}]interface{})
				manifest["metadata"].(map[string]interface{})["name"] = metadata["name"]
			}

			if value, ok := stageYaml["labels"]; ok {
				for k, v := range value.(map[interface{}]interface{}) {
					manifest["spec"].
					(map[string]interface{})["template"].
					(map[string]interface{})["metadata"].
					(map[string]interface{})["labels"].
					(map[string]interface{})[k.(string)] = v
				}
			}

			containers := manifest["spec"].
			(map[string]interface{})["template"].
			(map[string]interface{})["spec"].
			(map[string]interface{})["containers"].
			([]interface{})

			if parentStage == "worker" {
				gracefulStopCommand := []string{}
				workerType := stageYaml["type"].(string)
				if workerType == "shoryuken" || workerType == "sidekiq" {
					gracefulStopCommand = append(gracefulStopCommand, "/bin/kill", "-USR1", "1")
				} else if workerType == "delayedJob" {
					gracefulStopCommand = append(gracefulStopCommand, "/bin/kill", "-TERM", "1")
				}

				command := stageYaml["command"].(string)

				for _, c := range containers {
					args := c.(map[string]interface{})["args"].([]interface{})
					for i := range args {
						if args[i] == "WORKER_COMMAND" {
							args[i] = command
							c.(map[string]interface{})["lifecycle"].
							(map[string]interface{})["preStop"].
							(map[string]interface{})["exec"].
							(map[string]interface{})["command"] = gracefulStopCommand
							break
						}
					}
				}
			}

			for _, c := range containers {
				for _, image := range dockerImages {
					if strings.Contains(image, "amazonaws.com") { 
						repository := strings.Split(strings.Split(image, ":")[0], "/")[1]
						if c.(map[string]interface{})["image"] == repository {
							c.(map[string]interface{})["image"] = image
							break
						}
					}
				}
			}
		}

		if parentStage == "runJob" {
			cmds := []string{}
			for _, c := range stageYaml["commands"].([]interface{}) {
				cmds = append(cmds, c.(string))
			}
			command := strings.Join(cmds, " && ")
			container := stageJson["containers"].([]interface{})[0].(map[string]interface{})
			args := container["args"].([]interface{})
			for i := range args {
				if args[i] == "JOB_COMMAND" {
					args[i] = command
					break
				}
			}

			for _, image := range dockerImages {
				if strings.Contains(image, "amazonaws.com") { 
					tmpList := strings.Split(image, ":")
					tag := tmpList[1]
					tmpList = strings.Split(tmpList[0], "/")
					registry := tmpList[0]
					repository := tmpList[1]
					imageDescription := container["imageDescription"].(map[string]interface{})
					if imageDescription["repository"] == repository {
						imageDescription["imageId"] = image
						imageDescription["registry"] = registry
						imageDescription["tag"] = tag
						break
					}
				}
			}
		} else if parentStage == "runPipeline" {
			stageJson["pipeline"] = fmt.Sprintf("%s-%s", stageYaml["pipeline"], appAttributes["pipelineVersion"])
		}

		stages = append(stages, stageJson)
	}

	pipelineJson["stages"] = stages
	return pipelineJson, nil
}


func savePipeline(pipelineJson map[string]interface{}, flags *pflag.FlagSet) error {
	gateClient, err := gateclient.NewGateClient(flags)
	if err != nil {
		return err
	}

	valid := true
	if _, exists := pipelineJson["name"]; !exists {
		util.UI.Error("Required pipeline key 'name' missing...\n")
		valid = false
	}

	if _, exists := pipelineJson["application"]; !exists {
		util.UI.Error("Required pipeline key 'application' missing...\n")
		valid = false
	}

	if template, exists := pipelineJson["template"]; exists && len(template.(map[string]interface{})) > 0 {
		if _, exists := pipelineJson["schema"]; !exists {
			util.UI.Error("Required pipeline key 'schema' missing for templated pipeline...\n")
			valid = false
		}
	    pipelineJson["type"] = "templatedPipeline"
	}

	if !valid {
		return fmt.Errorf("Submitted pipeline is invalid: %s\n", pipelineJson)
	}
	application := pipelineJson["application"].(string)
	pipelineName := pipelineJson["name"].(string)

	foundPipeline, queryResp, _ := gateClient.ApplicationControllerApi.GetPipelineConfigUsingGET(gateClient.Context, application, pipelineName)

	if queryResp.StatusCode != http.StatusOK {
		return fmt.Errorf("Encountered an error querying pipeline, status code: %d\n", queryResp.StatusCode)
	}

	_, exists := pipelineJson["id"].(string)
	var foundPipelineId string
	if len(foundPipeline) > 0 {
		foundPipelineId = foundPipeline["id"].(string)
	}
	if !exists && foundPipelineId != "" {
		pipelineJson["id"] = foundPipelineId
	}

	saveResp, saveErr := gateClient.PipelineControllerApi.SavePipelineUsingPOST(gateClient.Context, pipelineJson)

	if saveErr != nil {
		fmt.Printf("s err: %v", saveErr)
		return err
	}
	if saveResp.StatusCode != http.StatusOK {
		return fmt.Errorf("Encountered an error saving pipeline, status code: %d\n", saveResp.StatusCode)
	}

	util.UI.Info(util.Colorize().Color(fmt.Sprintf("[reset][bold][blue]Pipeline save succeeded: %s", pipelineName)))
	return nil
}
