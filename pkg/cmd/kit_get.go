/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/spf13/cobra"
)

func newKitGetCmd(rootCmdOptions *RootCmdOptions) *cobra.Command {
	impl := kitGetCommand{
		RootCmdOptions: rootCmdOptions,
	}

	cmd := cobra.Command{
		Use:   "get",
		Short: "Get defined Integration Kit",
		Long:  `Get defined Integration Kit.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := impl.validate(cmd, args); err != nil {
				return err
			}
			if err := impl.run(); err != nil {
				fmt.Println(err.Error())
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&impl.user, v1alpha1.IntegrationKitTypeUser, true, "Includes user kits")
	cmd.Flags().BoolVar(&impl.external, v1alpha1.IntegrationKitTypeExternal, true, "Includes external kits")
	cmd.Flags().BoolVar(&impl.platform, v1alpha1.IntegrationKitTypePlatform, true, "Includes platform kits")

	return &cmd
}

type kitGetCommand struct {
	*RootCmdOptions
	user     bool
	external bool
	platform bool
}

func (command *kitGetCommand) validate(cmd *cobra.Command, args []string) error {
	return nil
}

func (command *kitGetCommand) run() error {
	kitList := v1alpha1.NewIntegrationKitList()
	c, err := command.GetCmdClient()
	if err != nil {
		return err
	}
	if err := c.List(command.Context, &k8sclient.ListOptions{Namespace: command.Namespace}, &kitList); err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', 0)
	fmt.Fprintln(w, "NAME\tPHASE\tTYPE\tIMAGE")
	for _, ctx := range kitList.Items {
		t := ctx.Labels["camel.apache.org/kit.type"]
		u := command.user && t == v1alpha1.IntegrationKitTypeUser
		e := command.external && t == v1alpha1.IntegrationKitTypeExternal
		p := command.platform && t == v1alpha1.IntegrationKitTypePlatform

		if u || e || p {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", ctx.Name, string(ctx.Status.Phase), t, ctx.Status.Image)
		}
	}
	w.Flush()

	return nil
}
