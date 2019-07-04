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
	"github.com/spinctl/cmd/gateclient"
	"github.com/spinctl/cmd/orca-tasks"
	"github.com/spinctl/cmd/pipeline"
	"github.com/spinctl/util"
	"strings"
)

type SaveOptions struct {
	*applicationOptions
	applicationFile string
}

var (
	saveApplicationShort   = "Save the provided application"
	saveApplicationLong    = "Save the specified application"
)

func NewSaveCmd(appOptions applicationOptions) *cobra.Command {
	options := SaveOptions{
		applicationOptions: &appOptions,
	}
	cmd := &cobra.Command{
		Use:     "save",
		Short:   saveApplicationShort,
		Long:    saveApplicationLong,
		RunE: func(cmd *cobra.Command, args []string) error {
			return saveApplication(cmd, options)
		},
	}
	cmd.PersistentFlags().StringVarP(&options.applicationFile, "file", "", "", "path to the application file")

	return cmd
}

func saveApplication(cmd *cobra.Command, options SaveOptions) error {
	// TODO(jacobkiefer): Should we check for an existing application of the same name?
	gateClient, err := gateclient.NewGateClient(cmd.InheritedFlags())
	if err != nil {
		return err
	}

	config, err := util.ParseYamlFromFile(options.applicationFile, true)
	if err != nil {
		return fmt.Errorf("Could not parse supplied application: %v.\n", err)
	}

	appAttributes := config["application"].(map[interface{}]interface{})

	var app map[string]interface{}
	if appAttributes["name"] == "" || appAttributes["ownerEmail"] == "" {
		return errors.New("Required application parameter missing, exiting...")
	}

	var cloudProviders []string
	for _, v := range appAttributes["cloudProviders"].([]interface{}) {
		cloudProviders = append(cloudProviders, v.(string))
	}

	app = map[string]interface{}{
		"cloudProviders": strings.Join(cloudProviders, ","),
		"instancePort":   80,
		"name":           appAttributes["name"],
		"email":          appAttributes["ownerEmail"],
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

	util.UI.Info(util.Colorize().Color(fmt.Sprintf("[reset][bold][green]Application save succeeded")))

	
	return nil
}

// func applicationExists(cmd *cobra.Command, applicationName string) (bool, error) {
// 	gateClient, err := gateclient.NewGateClient(cmd.InheritedFlags())
// 	if err != nil {
// 		return false, err
// 	}

// 	app, resp, err := gateClient.ApplicationControllerApi.GetApplicationUsingGET(gateClient.Context, applicationName, map[string]interface{}{"expand": false})
// 	if resp != nil {
// 		if resp.StatusCode == http.StatusNotFound {
// 			return false, nil
// 		} else if resp.StatusCode != http.StatusOK {
// 			return fmt.Errorf("Encountered an error getting application, status code: %d\n", resp.StatusCode)
// 		}
// 	}

// 	if err != nil {
// 		return false, err
// 	}

// 	return true, nil
// }
