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
	"github.com/spf13/cobra"
	"io"
)

type executionOptions struct{}

var (
	executionShort   = ""
	executionLong    = ""
	executionExample = ""
)

func NewExecutionCmd(out io.Writer) *cobra.Command {
	options := executionOptions{}
	cmd := &cobra.Command{
		Use:     "execution",
		Aliases: []string{"executions", "ex"},
		Short:   executionShort,
		Long:    executionLong,
		Example: executionExample,
	}

	// create subcommands
	cmd.AddCommand(NewCancelCmd())
	cmd.AddCommand(NewGetCmd(options))
	cmd.AddCommand(NewListCmd(options))
	return cmd
}
