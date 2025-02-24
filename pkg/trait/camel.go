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

package trait

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/defaults"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/maven"
	"github.com/pkg/errors"

	yaml2 "gopkg.in/yaml.v2"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
)

type camelTrait struct {
	BaseTrait      `property:",squash"`
	Version        string `property:"version"`
	RuntimeVersion string `property:"runtime-version"`
}

func newCamelTrait() *camelTrait {
	return &camelTrait{
		BaseTrait: newBaseTrait("camel"),
	}
}

func (t *camelTrait) Configure(e *Environment) (bool, error) {
	if t.Enabled != nil && !*t.Enabled {
		return false, nil
	}

	return true, nil
}

func (t *camelTrait) Apply(e *Environment) error {
	ns := e.DetermineNamespace()
	if ns == "" {
		return errors.New("unable to determine namespace")
	}

	cv := e.DetermineCamelVersion()
	rv := e.DetermineRuntimeVersion()

	if t.Version != "" {
		cv = t.Version
	}
	if t.RuntimeVersion != "" {
		rv = t.RuntimeVersion
	}

	if e.CamelCatalog == nil {
		c, err := camel.LoadCatalog(e.C, e.Client, ns, cv, rv)
		if err != nil {
			return err
		}
		if c == nil {
			// if the catalog is not found in the cluster, try to create it if
			// the required versions (camel and runtime) are not expressed as
			// semver constraints
			cvHasFixedVersion, err := regexp.MatchString(`^(\d+)\.(\d+)\.([\w-\.]+)$`, cv)
			if err != nil {
				return err
			}
			rvHasFixedVersion, err := regexp.MatchString(`^(\d+)\.(\d+)\.([\w-\.]+)$`, rv)
			if err != nil {
				return err
			}

			if cvHasFixedVersion && rvHasFixedVersion {
				c, err = t.GenerateCatalog(e, cv, rv)
				if err != nil {
					return err
				}

				// sanitize catalog name
				catalogName := "camel-catalog-" + strings.ToLower(cv+"-"+rv)

				cx := v1alpha1.NewCamelCatalogWithSpecs(ns, catalogName, c.CamelCatalogSpec)
				cx.Labels = make(map[string]string)
				cx.Labels["app"] = "camel-k"
				cx.Labels["camel.apache.org/catalog.version"] = cv
				cx.Labels["camel.apache.org/catalog.loader.version"] = cv
				cx.Labels["camel.apache.org/runtime.version"] = rv
				cx.Labels["camel.apache.org/catalog.generated"] = True

				err = e.Client.Create(e.C, &cx)
				if err != nil && !k8serrors.IsAlreadyExists(err) {
					return err
				}
			}
		}

		if c == nil {
			return fmt.Errorf("unable to find catalog for: %s", cv)
		}

		e.CamelCatalog = c
	}

	e.RuntimeVersion = rv

	if e.Integration != nil {
		e.Integration.Status.CamelVersion = e.CamelCatalog.Version
		e.Integration.Status.RuntimeVersion = rv
	}
	if e.IntegrationKit != nil {
		e.IntegrationKit.Status.CamelVersion = e.CamelCatalog.Version
		e.IntegrationKit.Status.RuntimeVersion = rv
	}

	return nil
}

// GenerateCatalog --
func (t *camelTrait) GenerateCatalog(e *Environment, camelVersion string, runtimeVersion string) (*camel.RuntimeCatalog, error) {
	root := os.TempDir()
	tmpDir, err := ioutil.TempDir(root, "camel-catalog")
	if err != nil {
		return nil, err
	}

	defer os.RemoveAll(tmpDir)

	if err := os.MkdirAll(tmpDir, os.ModePerm); err != nil {
		return nil, err
	}

	project, err := t.GenerateMavenProject(camelVersion, runtimeVersion)
	if err != nil {
		return nil, err
	}

	mc := maven.NewContext(tmpDir, project)
	mc.LocalRepository = e.Platform.Spec.Build.LocalRepository
	mc.Timeout = e.Platform.Spec.Build.Maven.Timeout.Duration
	mc.AddSystemProperty("catalog.path", tmpDir)
	mc.AddSystemProperty("catalog.file", "catalog.yaml")

	ns := e.DetermineNamespace()
	if ns == "" {
		return nil, errors.New("unable to determine namespace")
	}

	settings, err := kubernetes.ResolveValueSource(e.C, e.Client, ns, &e.Platform.Spec.Build.Maven.Settings)
	if err != nil {
		return nil, err
	}
	if settings != "" {
		mc.SettingsContent = []byte(settings)
	}

	err = maven.Run(mc)
	if err != nil {
		return nil, err
	}

	content, err := ioutil.ReadFile(path.Join(tmpDir, "catalog.yaml"))
	if err != nil {
		return nil, err
	}

	catalog := v1alpha1.CamelCatalog{}
	if err := yaml2.Unmarshal(content, &catalog); err != nil {
		return nil, err
	}

	return camel.NewRuntimeCatalog(catalog.Spec), nil
}

// GenerateCatalogMavenProject --
func (t *camelTrait) GenerateMavenProject(camelVersion string, runtimeVersion string) (maven.Project, error) {
	p := maven.NewProjectWithGAV("org.apache.camel.k.integration", "camel-k-catalog-generator", defaults.Version)
	p.Build = &maven.Build{
		DefaultGoal: "generate-resources",
		Plugins: []maven.Plugin{
			{
				GroupID:    "org.apache.camel.k",
				ArtifactID: "camel-k-maven-plugin",
				Version:    runtimeVersion,
				Executions: []maven.Execution{
					{
						ID: "generate-catalog",
						Goals: []string{
							"generate-catalog",
						},
					},
				},
				Dependencies: []maven.Dependency{
					{
						GroupID:    "org.apache.camel",
						ArtifactID: "camel-catalog",
						Version:    camelVersion,
					},
				},
			},
		},
	}

	return p, nil
}
