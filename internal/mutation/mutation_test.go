/*
Copyright 2025 The KCP Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package mutation

import (
	"encoding/json"
	"testing"

	syncagentv1alpha1 "github.com/kcp-dev/api-syncagent/sdk/apis/syncagent/v1alpha1"
)

func TestApplyResourceMutation(t *testing.T) {
	testcases := []struct {
		name      string
		inputData string
		mutation  syncagentv1alpha1.ResourceMutation
		ctx       *TemplateMutationContext
		expected  string
	}{
		// regex

		{
			name:      "regex: replace one existing value",
			inputData: `{"spec":{"secretName":"foo"}}`,
			mutation: syncagentv1alpha1.ResourceMutation{
				Regex: &syncagentv1alpha1.ResourceRegexMutation{
					Path:        "spec.secretName",
					Pattern:     "",
					Replacement: "new-value",
				},
			},
			expected: `{"spec":{"secretName":"new-value"}}`,
		},
		{
			name:      "regex: rewrite one existing value",
			inputData: `{"spec":{"secretName":"foo"}}`,
			mutation: syncagentv1alpha1.ResourceMutation{
				Regex: &syncagentv1alpha1.ResourceRegexMutation{
					Path:        "spec.secretName",
					Pattern:     "o",
					Replacement: "u",
				},
			},
			expected: `{"spec":{"secretName":"fuu"}}`,
		},
		{
			name:      "regex: should support grouping",
			inputData: `{"spec":{"secretName":"foo"}}`,
			mutation: syncagentv1alpha1.ResourceMutation{
				Regex: &syncagentv1alpha1.ResourceRegexMutation{
					Path:        "spec.secretName",
					Pattern:     "(f)oo",
					Replacement: "oo$1",
				},
			},
			expected: `{"spec":{"secretName":"oof"}}`,
		},
		{
			name:      "regex: coalesces to strings",
			inputData: `{"spec":{"aNumber":24}}`,
			mutation: syncagentv1alpha1.ResourceMutation{
				Regex: &syncagentv1alpha1.ResourceRegexMutation{
					Path:        "spec.aNumber",
					Pattern:     "4",
					Replacement: "5",
				},
			},
			expected: `{"spec":{"aNumber":"25"}}`,
		},
		{
			name:      "regex: can change types",
			inputData: `{"spec":{"aNumber":24}}`,
			mutation: syncagentv1alpha1.ResourceMutation{
				Regex: &syncagentv1alpha1.ResourceRegexMutation{
					Path:        "spec",
					Replacement: "new-value",
				},
			},
			expected: `{"spec":"new-value"}`,
		},
		{
			name:      "regex: can change types /2",
			inputData: `{"spec":{"aNumber":24}}`,
			mutation: syncagentv1alpha1.ResourceMutation{
				Regex: &syncagentv1alpha1.ResourceRegexMutation{
					Path: "spec",
					// Due to the string coalescing, this will turn the {aNumber:42} object
					// into a string, of which we match every character and return it,
					// effectively stringify-ing an object.
					Pattern:     "(.)",
					Replacement: "$1",
				},
			},
			expected: `{"spec":"{\"aNumber\":24}"}`,
		},
		{
			name:      "regex: can empty values",
			inputData: `{"spec":{"aNumber":24}}`,
			mutation: syncagentv1alpha1.ResourceMutation{
				Regex: &syncagentv1alpha1.ResourceRegexMutation{
					Path:        "spec",
					Replacement: "",
				},
			},
			expected: `{"spec":""}`,
		},
		{
			name:      "regex: can empty values /2",
			inputData: `{"spec":{"aNumber":24}}`,
			mutation: syncagentv1alpha1.ResourceMutation{
				Regex: &syncagentv1alpha1.ResourceRegexMutation{
					Path:        "spec",
					Pattern:     ".+",
					Replacement: "",
				},
			},
			expected: `{"spec":""}`,
		},

		// templates

		{
			name:      "template: empty template returns empty value",
			inputData: `{"spec":{"secretName":"foo"}}`,
			mutation: syncagentv1alpha1.ResourceMutation{
				Template: &syncagentv1alpha1.ResourceTemplateMutation{
					Path: "spec.secretName",
				},
			},
			expected: `{"spec":{"secretName":""}}`,
		},
		{
			name:      "template: can change value type",
			inputData: `{"spec":{"secretName":"foo"}}`,
			mutation: syncagentv1alpha1.ResourceMutation{
				Template: &syncagentv1alpha1.ResourceTemplateMutation{
					Path: "spec",
				},
			},
			expected: `{"spec":""}`,
		},
		{
			name:      "template: execute basic template",
			inputData: `{"spec":{"secretName":"foo"}}`,
			mutation: syncagentv1alpha1.ResourceMutation{
				Template: &syncagentv1alpha1.ResourceTemplateMutation{
					Path:     "spec.secretName",
					Template: `{{ upper .Value.String }}`,
				},
			},
			expected: `{"spec":{"secretName":"FOO"}}`,
		},

		// delete

		{
			name:      "delete: can remove object keys",
			inputData: `{"spec":{"secretName":"foo"}}`,
			mutation: syncagentv1alpha1.ResourceMutation{
				Delete: &syncagentv1alpha1.ResourceDeleteMutation{
					Path: "spec.secretName",
				},
			},
			expected: `{"spec":{}}`,
		},
		{
			name:      "delete: can remove array items",
			inputData: `{"spec":[1,2,3]}`,
			mutation: syncagentv1alpha1.ResourceMutation{
				Delete: &syncagentv1alpha1.ResourceDeleteMutation{
					Path: "spec.1",
				},
			},
			expected: `{"spec":[1,3]}`,
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			// encode current value as JSON
			var inputData any
			if err := json.Unmarshal([]byte(testcase.inputData), &inputData); err != nil {
				t.Fatalf("Failed to JSON encode input data: %v", err)
			}

			mutated, err := ApplyResourceMutation(inputData, testcase.mutation, testcase.ctx)
			if err != nil {
				t.Fatalf("Function returned unexpected error: %v", err)
			}

			result, err := json.Marshal(mutated)
			if err != nil {
				t.Fatalf("Failed to JSON encode output: %v", err)
			}

			output := string(result)
			if testcase.expected != output {
				t.Errorf("Expected %q, but got %q.", testcase.expected, output)
			}
		})
	}
}
