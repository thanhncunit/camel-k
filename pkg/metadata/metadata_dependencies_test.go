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

package metadata

import (
	"testing"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/test"

	"github.com/stretchr/testify/assert"
)

func TestDependenciesJavaSource(t *testing.T) {
	code := v1alpha1.SourceSpec{
		DataSpec: v1alpha1.DataSpec{
			Name: "Request.java",
			Content: `
			    from("telegram:bots/cippa").to("log:stash");
			    from("timer:tick").to("amqp:queue");
			    from("ine:xistent").to("amqp:queue");
				from("twitter-search:{{twitterKeywords}}"
                    + "?delay={{twitterDelayMs}}");
			`,
		},
		Language: v1alpha1.LanguageJavaSource,
	}

	catalog, err := test.DefaultCatalog()
	assert.Nil(t, err)

	meta := Extract(catalog, code)

	assert.ElementsMatch(t, []string{"camel:amqp", "camel:log", "camel:telegram", "camel:timer", "camel:twitter"}, meta.Dependencies.List())
}

func TestDependenciesJavaScript(t *testing.T) {
	code := v1alpha1.SourceSpec{
		DataSpec: v1alpha1.DataSpec{
			Name: "source.js",
			Content: `
			    from('telegram:bots/cippa').to("log:stash");
			    from('timer:tick').to("amqp:queue");
			    from("ine:xistent").to("amqp:queue");
			    '"'
		    `,
		},
		Language: v1alpha1.LanguageJavaScript,
	}

	catalog, err := test.DefaultCatalog()
	assert.Nil(t, err)

	meta := Extract(catalog, code)

	assert.ElementsMatch(t, []string{"camel:amqp", "camel:log", "camel:telegram", "camel:timer"}, meta.Dependencies.List())
}

func TestDependenciesGroovy(t *testing.T) {
	code := v1alpha1.SourceSpec{
		DataSpec: v1alpha1.DataSpec{
			Name: "source.groovy",
			Content: `
			    from('telegram:bots/cippa').to("log:stash");
			    from('timer:tick').to("amqp:queue");
			    from("ine:xistent").to("amqp:queue");
				from('twitter-search:{{twitterKeywords}}'
                    + '?delay={{twitterDelayMs}}');
			    '"
		    `,
		},
		Language: v1alpha1.LanguageGroovy,
	}

	catalog, err := test.DefaultCatalog()
	assert.Nil(t, err)

	meta := Extract(catalog, code)

	assert.ElementsMatch(t, []string{"camel:amqp", "camel:log", "camel:telegram", "camel:timer", "camel:twitter"}, meta.Dependencies.List())
}

func TestDependencies(t *testing.T) {
	code := v1alpha1.SourceSpec{
		DataSpec: v1alpha1.DataSpec{
			Name: "Request.java",
			Content: `
			    from("http:test").to("log:end");
			    from("https4:test").to("log:end");
			    from("twitter-timeline:test").to("mock:end");
		    `,
		},
		Language: v1alpha1.LanguageJavaSource,
	}

	catalog, err := test.DefaultCatalog()
	assert.Nil(t, err)

	meta := Extract(catalog, code)

	assert.ElementsMatch(t, []string{"camel:http", "camel:log", "camel:mock", "camel:twitter"}, meta.Dependencies.List())
}

func TestJacksonDependency(t *testing.T) {
	code := v1alpha1.SourceSpec{
		DataSpec: v1alpha1.DataSpec{
			Name: "Request.java",
			Content: `
			    from("http:test").unmarshal().json(JsonLibrary.Jackson).to("log:end");
		    `,
		},
		Language: v1alpha1.LanguageJavaSource,
	}

	catalog, err := test.DefaultCatalog()
	assert.Nil(t, err)

	meta := Extract(catalog, code)

	assert.ElementsMatch(t, []string{"camel:http", "camel:jackson", "camel:log"}, meta.Dependencies.List())
}

func TestLanguageDependencies(t *testing.T) {
	code := v1alpha1.SourceSpec{
		DataSpec: v1alpha1.DataSpec{
			Name: "Languages.java",
			Content: `
				from("direct:start")
				.transform().ognl("request.body.name == 'Camel K'")
				.transform().simple("${body.toUpperCase()}")
				.transform().mvel("resource:classpath:script.mvel")
				.transform().xquery("/ns:foo/bar", String.class, new Namespaces("ns", "http://foo/bar"))
				.transform().xpath("//foo/bar")
				.transform().jsonpath("$.foo")
				.transform().groovy("request.body += 'modified'")
				.split().xtokenize("/ns:foo/bar", new Namespaces("ns", "http://foo/bar"));
			`,
		},
		Language: v1alpha1.LanguageJavaSource,
	}

	catalog, err := test.DefaultCatalog()
	assert.Nil(t, err)

	meta := Extract(catalog, code)
	assert.ElementsMatch(t, []string{"camel:direct", "camel:bean", "camel:ognl", "camel:saxon", "camel:xpath",
		"camel:jsonpath", "camel:groovy", "camel:jaxp", "camel:mvel"}, meta.Dependencies.List())
}

func TestLanguageDependenciesTransformExpression(t *testing.T) {
	code := v1alpha1.SourceSpec{
		DataSpec: v1alpha1.DataSpec{
			Name: "Languages.java",
			Content: `
				from("direct:start")
				.transform(language("ognl", "request.body.name == 'Camel K'"))
				.transform(simple("${body.toUpperCase()}"))
				.transform(xpath("//foo/bar"))
				.transform(jsonpath("$.foo"))
			`,
		},
		Language: v1alpha1.LanguageJavaSource,
	}

	catalog, err := test.DefaultCatalog()
	assert.Nil(t, err)

	meta := Extract(catalog, code)
	assert.ElementsMatch(t, []string{"camel:direct", "camel:bean", "camel:ognl", "camel:xpath", "camel:jsonpath"}, meta.Dependencies.List())
}

func TestHystrixDependency(t *testing.T) {
	code := v1alpha1.SourceSpec{
		DataSpec: v1alpha1.DataSpec{
			Name: "Request.groovy",
			Content: `
			    from("http:test")
					.hystrix()
						.to("log:end")
					.onFallback()
						.to("log:fallback")
		    `,
		},
		Language: v1alpha1.LanguageGroovy,
	}

	catalog, err := test.DefaultCatalog()
	assert.Nil(t, err)

	meta := Extract(catalog, code)

	assert.ElementsMatch(t, []string{"camel:http", "camel:hystrix", "camel:log"}, meta.Dependencies.List())
}

func TestRestDependency(t *testing.T) {
	code := v1alpha1.SourceSpec{
		DataSpec: v1alpha1.DataSpec{
			Name: "Request.groovy",
			Content: `
                rest()
                    .get("/api")
                    .to("direct:get")
			    from("http:test")
                    .to("log:info")
		    `,
		},
		Language: v1alpha1.LanguageGroovy,
	}

	catalog, err := test.DefaultCatalog()
	assert.Nil(t, err)

	meta := Extract(catalog, code)

	assert.ElementsMatch(t, []string{"camel:http", "camel:rest", "camel:direct", "camel:log"}, meta.Dependencies.List())
}

func TestRestWithPathDependency(t *testing.T) {
	code := v1alpha1.SourceSpec{
		DataSpec: v1alpha1.DataSpec{
			Name: "Request.groovy",
			Content: `
                rest("/test")
                    .get("/api")
                    .to("direct:get")
			    from("http:test")
                    .to("log:info")
		    `,
		},
		Language: v1alpha1.LanguageGroovy,
	}

	catalog, err := test.DefaultCatalog()
	assert.Nil(t, err)

	meta := Extract(catalog, code)

	assert.ElementsMatch(t, []string{"camel:http", "camel:rest", "camel:direct", "camel:log"}, meta.Dependencies.List())
}

func TestRestConfigurationDependency(t *testing.T) {
	code := v1alpha1.SourceSpec{
		DataSpec: v1alpha1.DataSpec{
			Name: "Request.groovy",
			Content: `
                restConfiguration()
                    .component("undertow")
			    from("http:test")
                    .to("log:info")
		    `,
		},
		Language: v1alpha1.LanguageGroovy,
	}

	catalog, err := test.DefaultCatalog()
	assert.Nil(t, err)

	meta := Extract(catalog, code)

	assert.ElementsMatch(t, []string{"camel:http", "camel:rest", "camel:log"}, meta.Dependencies.List())
}

func TestRestClosureDependency(t *testing.T) {
	code := v1alpha1.SourceSpec{
		DataSpec: v1alpha1.DataSpec{
			Name: "Request.groovy",
			Content: `
                rest {
                }
			    from("http:test")
                    .to("log:info")
		    `,
		},
		Language: v1alpha1.LanguageGroovy,
	}

	catalog, err := test.DefaultCatalog()
	assert.Nil(t, err)

	meta := Extract(catalog, code)

	assert.ElementsMatch(t, []string{"camel:http", "camel:rest", "camel:log"}, meta.Dependencies.List())
}

func TestXMLHystrixDependency(t *testing.T) {
	code := v1alpha1.SourceSpec{

		DataSpec: v1alpha1.DataSpec{
			Name: "routes.xml",
			Content: `
			<from uri="direct:ciao" />
			<hystrix>
				<to uri="log:info" />
				<onFallback>
					<to uri="kafka:topic" />
				</onFallback>
			</hystrix>
		`,
		},
		Language: v1alpha1.LanguageXML,
	}

	catalog, err := test.DefaultCatalog()
	assert.Nil(t, err)

	meta := Extract(catalog, code)

	assert.ElementsMatch(t, []string{"camel:direct", "camel:hystrix", "camel:kafka", "camel:log"}, meta.Dependencies.List())
}

func TestXMLRestDependency(t *testing.T) {
	code := v1alpha1.SourceSpec{

		DataSpec: v1alpha1.DataSpec{
			Name: "routes.xml",
			Content: `
			<rest path="/say">
		      <get uri="/hello">
		        <to uri="direct:hello"/>
		      </get>
		      <get uri="/bye" consumes="application/json">
		        <to uri="direct:bye"/>
		      </get>
		      <post uri="/bye">
		        <to uri="mock:update"/>
		      </post>
		    </rest>
		`,
		},
		Language: v1alpha1.LanguageXML,
	}

	catalog, err := test.DefaultCatalog()
	assert.Nil(t, err)

	meta := Extract(catalog, code)

	assert.ElementsMatch(t, []string{"camel:direct", "camel:rest", "camel:mock"}, meta.Dependencies.List())
}

func TestXMLLanguageDependencies(t *testing.T) {
	code := v1alpha1.SourceSpec{
		DataSpec: v1alpha1.DataSpec{
			Name: "routes.xml",
			Content: `
               <from uri="direct:start" />
               <transform>
                 <language language="ognl">request.body.name == 'Camel K'</language>
               </transform>
               <transform>
                 <simple>${body.toUpperCase()}</simple>
               </transform>
               <transform>
                 <mvel>resource:classpath:script.mvel</mvel>
               </transform>
               <transform>
                 <jsonpath>$.foo</jsonpath>
               </transform>
               <transform>
                 <groovy>request.body += 'modified'</groovy>
               </transform>
               <transform>
                 <tokenize>request.body += 'modified'</tokenize>
               </transform>
               <transform>
                 <xtokenize>/ns:foo/bar</xtokenize>
               </transform>
               <transform>
                 <xpath>//foo/bar</xpath>
               </transform>
               <transform>
                 <xquery>//ns:foo/bar</xquery>
               </transform>
               <split>
                 <tokenize token=","/>
               </split>
          `,
		},
		Language: v1alpha1.LanguageXML,
	}

	catalog, err := test.DefaultCatalog()
	assert.Nil(t, err)

	meta := Extract(catalog, code)
	assert.ElementsMatch(t, []string{"camel:direct", "camel:bean", "camel:ognl", "camel:saxon", "camel:xpath",
		"camel:jsonpath", "camel:groovy", "camel:jaxp", "camel:mvel"}, meta.Dependencies.List())
}

const yamlWithRest = `
- rest:
    path: "/"
    steps:
        - to: "log:info"
        - to: "direct:hello"
`
const yamlWithHystrix = `
- from:
    uri: "direct:start"
    steps:
        - hystrix:
            todo: "not implemented"
`

const yamlWithLanguages = `
- from:
    uri: "direct:start"
    steps:
        - set-body:
            constant: "Hello Camel K"
        - transform:
            language:
                language: "ognl"
                expression: "request.body.name == 'Camel K'"
        - transform:
            simple: "${body.toUpperCase()}"
        - transform:
            mvel: "resource:classpath:script.mvel"
        - transform:
            xquery: "/ns:foo/bar"
        - transform:
            xpath: "//foo/bar"
        - transform:
            jsonpath: "$.foo"
        - transform:
            groovy: "request.body += 'modified'"
        - split:
            xtokenize: "/ns:foo/bar"
`

func TestYAMLRestDependency(t *testing.T) {
	code := v1alpha1.SourceSpec{

		DataSpec: v1alpha1.DataSpec{
			Name:    "routes.yaml",
			Content: yamlWithRest,
		},
		Language: v1alpha1.LanguageYaml,
	}

	catalog, err := test.DefaultCatalog()
	assert.Nil(t, err)

	meta := Extract(catalog, code)

	assert.ElementsMatch(t, []string{"camel:direct", "camel:rest", "camel:log"}, meta.Dependencies.List())
}

func TestYAMLHystrixDependency(t *testing.T) {
	code := v1alpha1.SourceSpec{

		DataSpec: v1alpha1.DataSpec{
			Name:    "routes.yaml",
			Content: yamlWithHystrix,
		},
		Language: v1alpha1.LanguageYaml,
	}

	catalog, err := test.DefaultCatalog()
	assert.Nil(t, err)

	meta := Extract(catalog, code)

	assert.ElementsMatch(t, []string{"camel:direct", "camel:hystrix"}, meta.Dependencies.List())
}

func TestYAMLLanguageDependencies(t *testing.T) {
	code := v1alpha1.SourceSpec{
		DataSpec: v1alpha1.DataSpec{
			Name:    "routes.yaml",
			Content: yamlWithLanguages,
		},
		Language: v1alpha1.LanguageYaml,
	}

	catalog, err := test.DefaultCatalog()
	assert.Nil(t, err)

	meta := Extract(catalog, code)

	assert.ElementsMatch(t, []string{"camel:direct", "camel:bean", "camel:ognl", "camel:saxon", "camel:xpath",
		"camel:jsonpath", "camel:groovy", "camel:jaxp", "camel:mvel"}, meta.Dependencies.List())
}
