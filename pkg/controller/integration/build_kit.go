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

package integration

import (
	"context"
	"fmt"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/trait"
	"github.com/apache/camel-k/pkg/util"
	"github.com/rs/xid"
)

// NewBuildKitAction create an action that handles integration kit build
func NewBuildKitAction() Action {
	return &buildKitAction{}
}

type buildKitAction struct {
	baseAction
}

func (action *buildKitAction) Name() string {
	return "build-kit"
}

func (action *buildKitAction) CanHandle(integration *v1alpha1.Integration) bool {
	return integration.Status.Phase == v1alpha1.IntegrationPhaseBuildingKit ||
		integration.Status.Phase == v1alpha1.IntegrationPhaseResolvingKit
}

func (action *buildKitAction) Handle(ctx context.Context, integration *v1alpha1.Integration) (*v1alpha1.Integration, error) {
	kit, err := LookupKitForIntegration(ctx, action.client, integration)
	if err != nil {
		//TODO: we may need to add a wait strategy, i.e give up after some time
		return nil, err
	}

	if kit != nil {
		if kit.Labels["camel.apache.org/kit.type"] == v1alpha1.IntegrationKitTypePlatform {
			// This is a platform kit and as it is auto generated it may get
			// out of sync if the integration that has generated it, has been
			// amended to add/remove dependencies

			versionMatch := kit.Status.Version == integration.Status.Version

			//TODO: this is a very simple check, we may need to provide a deps comparison strategy
			dependenciesMatch := util.StringSliceContains(kit.Spec.Dependencies, integration.Status.Dependencies)

			if !dependenciesMatch || !versionMatch {
				// We need to re-generate a kit or search for a new one that
				// satisfies integrations needs so let's remove the association
				// with a kit
				integration.SetIntegrationKit(&v1alpha1.IntegrationKit{})

				return integration, nil
			}
		}

		if kit.Status.Phase == v1alpha1.IntegrationKitPhaseError {
			integration.Status.Image = kit.Status.Image
			integration.Status.Phase = v1alpha1.IntegrationPhaseError
			integration.SetIntegrationKit(kit)

			return integration, nil
		}

		if kit.Status.Phase == v1alpha1.IntegrationKitPhaseReady {
			integration.Status.Image = kit.Status.Image
			integration.SetIntegrationKit(kit)

			if _, err := trait.Apply(ctx, action.client, integration, kit); err != nil {
				return nil, err
			}

			return integration, nil
		}

		if integration.Status.Kit == "" {
			integration.SetIntegrationKit(kit)

			return integration, nil
		}

		return nil, nil
	}

	platformKitName := fmt.Sprintf("kit-%s", xid.New())
	platformKit := v1alpha1.NewIntegrationKit(integration.Namespace, platformKitName)

	// Add some information for post-processing, this may need to be refactored
	// to a proper data structure
	platformKit.Labels = map[string]string{
		"camel.apache.org/kit.type":               v1alpha1.IntegrationKitTypePlatform,
		"camel.apache.org/kit.created.by.kind":    v1alpha1.IntegrationKind,
		"camel.apache.org/kit.created.by.name":    integration.Name,
		"camel.apache.org/kit.created.by.version": integration.ResourceVersion,
	}

	// Set the kit to have the same characteristics as the integrations
	platformKit.Spec = v1alpha1.IntegrationKitSpec{
		Dependencies: integration.Status.Dependencies,
		Repositories: integration.Spec.Repositories,
		Traits:       integration.Spec.Traits,
	}

	if err := action.client.Create(ctx, &platformKit); err != nil {
		return nil, err
	}

	// Set the kit name so the next handle loop, will fall through the
	// same path as integration with a user defined kit
	integration.SetIntegrationKit(&platformKit)

	return integration, nil
}
