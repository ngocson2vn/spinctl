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

package application

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spinctl/cmd/gateclient"
	"github.com/spinctl/cmd/orca-tasks"
	"github.com/spinctl/cmd/pipeline"
	"github.com/spinctl/util"
	"strings"
	"time"
	"path/filepath"
)

type DeployOptions struct {
	*applicationOptions
	applicationFile string
	dockerImages []string
	pipelineVersion string 
	timestamp string
	timeout int
	dryRun bool
}

var (
	saveApplicationShort   = "Save the provided application"
	saveApplicationLong    = "Save the specified application"
)

func NewDeployCmd(appOptions applicationOptions) *cobra.Command {
	tm := time.Now()
	year, month, day := tm.Date()
	hour, minute, second := tm.Clock()
	ts := tm.UnixNano() / 1e6

	options := DeployOptions{
		applicationOptions: &appOptions,
		pipelineVersion: fmt.Sprintf("%d%02d%02d-%02d%02d%02d", year, month, day, hour, minute, second),
		timestamp: fmt.Sprintf("%d", ts),
	}
	cmd := &cobra.Command{
		Use:     "deploy",
		Short:   saveApplicationShort,
		Long:    saveApplicationLong,
		RunE: func(cmd *cobra.Command, args []string) error {
			return save(cmd, options)
		},
	}
	cmd.PersistentFlags().StringVarP(&options.applicationFile, "file", "f", "", "path to the application file")
	cmd.PersistentFlags().StringArrayVarP(&options.dockerImages, "image", "i", []string{}, "path to the application file")
	cmd.PersistentFlags().IntVarP(&options.timeout, "timeout", "t", 1800, "path to the application file")
	cmd.PersistentFlags().BoolVarP(&options.dryRun, "dry-run", "d", false, "path to the application file")

	return cmd
}

func save(cmd *cobra.Command, options DeployOptions) error {
	flags := cmd.InheritedFlags()
	config, err := util.ParseYamlFromFile(options.applicationFile, true)
	if err != nil {
		return fmt.Errorf("Could not parse supplied application: %v.\n", err)
	}

	spinnakerApp := config["application"].(map[interface{}]interface{})
	err = saveApplication(spinnakerApp, flags)
	if err != nil {
		return err
	}

	err = savePipelines(spinnakerApp, options, flags)
	if err != nil {
		return err
	}

	err = executePipelines(spinnakerApp, options, flags)
	if err != nil {
		return err
	}

	return nil
}

func saveApplication(spinnakerApp map[interface{}]interface{}, flags *pflag.FlagSet) error {
	// TODO(jacobkiefer): Should we check for an existing application of the same name?
	gateClient, err := gateclient.NewGateClient(flags)
	if err != nil {
		return err
	}

	var app map[string]interface{}
	if spinnakerApp["name"] == "" || spinnakerApp["ownerEmail"] == "" {
		return errors.New("Required application parameter missing, exiting...")
	}

	var cloudProviders []string
	for _, v := range spinnakerApp["cloudProviders"].([]interface{}) {
		cloudProviders = append(cloudProviders, v.(string))
	}

	app = map[string]interface{}{
		"cloudProviders": strings.Join(cloudProviders, ","),
		"instancePort":   80,
		"name":           spinnakerApp["name"],
		"email":          spinnakerApp["ownerEmail"],
	}

	createAppTask := map[string]interface{}{
		"job":         []interface{}{map[string]interface{}{"type": "createApplication", "application": app}},
		"application": app["name"],
		"description": fmt.Sprintf("Create Application: %s", app["name"]),
	}

	ref, _, err := gateClient.TaskControllerApi.TaskUsingPOST1(gateClient.Context, createAppTask)
	if err != nil {
		return err
	}

	err = orca_tasks.WaitForSuccessfulTask(gateClient, ref, 5)
	if err != nil {
		return err
	}

	util.UI.Info(util.Colorize().Color(fmt.Sprintf("[reset][bold][blue]Application save succeeded: %s", app["name"])))

	return nil
}

func savePipelines(spinnakerApp map[interface{}]interface{}, options DeployOptions, flags *pflag.FlagSet) error {
	pipelineYamls := spinnakerApp["pipelines"].([]interface{})
	appAttributes := make(map[string]string)
	appAttributes["name"] = spinnakerApp["name"].(string)
	appAttributes["ownerEmail"] = spinnakerApp["name"].(string)
	appAttributes["slackChannel"] = spinnakerApp["slackChannel"].(string)
	appAttributes["spinnakerDir"] = filepath.Dir(options.applicationFile)
	appAttributes["pipelineVersion"] = options.pipelineVersion
	appAttributes["timestamp"] = options.timestamp

	var err error
	if options.dryRun {
		err = pipeline.SaveDryRun(pipelineYamls, appAttributes, options.dockerImages, flags)
	} else {
		err = pipeline.Save(pipelineYamls, appAttributes, options.dockerImages, flags)
	}

	if err != nil {
		return err
	}

	return nil
}

func executePipelines(spinnakerApp map[interface{}]interface{}, options DeployOptions, flags *pflag.FlagSet) error {
	applicationName := spinnakerApp["name"].(string)
	pipelineNames := []string{}
	for _, v := range spinnakerApp["pipelines"].([]interface{}) {
		p := v.(map[interface{}]interface{})
		if p["executable"] != nil && p["executable"].(bool) {
			pipelineNames = append(pipelineNames, fmt.Sprintf("%s-%s", p["name"].(string), options.pipelineVersion))
		}
	}

	for i := range pipelineNames {
		err := pipeline.Execute(applicationName, pipelineNames[i], flags)
		if err != nil {
			return err
		}
	}

	return nil
}
