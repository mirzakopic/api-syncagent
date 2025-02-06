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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"regexp"
	"strings"

	"github.com/Masterminds/sprig/v3"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	syncagentv1alpha1 "github.com/kcp-dev/api-syncagent/sdk/apis/syncagent/v1alpha1"
)

func ApplyResourceMutations(value any, mutations []syncagentv1alpha1.ResourceMutation, ctx *TemplateMutationContext) (any, error) {
	for _, mut := range mutations {
		var err error
		value, err = ApplyResourceMutation(value, mut, ctx)
		if err != nil {
			return nil, err
		}
	}

	return value, nil
}

func ApplyResourceMutation(value any, mut syncagentv1alpha1.ResourceMutation, ctx *TemplateMutationContext) (any, error) {
	// encode current value as JSON
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("failed to JSON encode value: %w", err)
	}

	// apply mutation
	jsonData, err := applyResourceMutationToJSON(string(encoded), mut, ctx)
	if err != nil {
		return nil, err
	}

	// decode back
	var result any
	err = json.Unmarshal([]byte(jsonData), &result)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}

	return result, nil
}

func applyResourceMutationToJSON(jsonData string, mut syncagentv1alpha1.ResourceMutation, ctx *TemplateMutationContext) (string, error) {
	switch {
	case mut.Delete != nil:
		return applyResourceDeleteMutation(jsonData, *mut.Delete)
	case mut.Template != nil:
		return applyResourceTemplateMutation(jsonData, *mut.Template, ctx)
	case mut.Regex != nil:
		return applyResourceRegexMutation(jsonData, *mut.Regex)
	default:
		return "", errors.New("must use either regex, template or delete mutation")
	}
}

func applyResourceDeleteMutation(jsonData string, mut syncagentv1alpha1.ResourceDeleteMutation) (string, error) {
	jsonData, err := sjson.Delete(jsonData, mut.Path)
	if err != nil {
		return "", fmt.Errorf("failed to delete value @ %s: %w", mut.Path, err)
	}

	return jsonData, nil
}

func applyResourceRegexMutation(jsonData string, mut syncagentv1alpha1.ResourceRegexMutation) (string, error) {
	if mut.Pattern == "" {
		return sjson.Set(jsonData, mut.Path, mut.Replacement)
	}

	// get the current value
	value := gjson.Get(jsonData, mut.Path)
	if !value.Exists() {
		return "", fmt.Errorf("path %s did not match any element in the document", mut.Path)
	}

	expr, err := regexp.Compile(mut.Pattern)
	if err != nil {
		return "", fmt.Errorf("invalid pattern %q: %w", mut.Pattern, err)
	}

	// this does apply some coalescing, like turning numbers into strings
	strVal := value.String()
	replacement := expr.ReplaceAllString(strVal, mut.Replacement)

	return sjson.Set(jsonData, mut.Path, replacement)
}

func templateFuncMap() template.FuncMap {
	funcs := sprig.TxtFuncMap()
	funcs["join"] = strings.Join
	return funcs
}

type TemplateMutationContext struct {
	// Value is always set by this package to the value found in the document.
	Value gjson.Result

	LocalObject  map[string]any
	RemoteObject map[string]any
}

func applyResourceTemplateMutation(jsonData string, mut syncagentv1alpha1.ResourceTemplateMutation, ctx *TemplateMutationContext) (string, error) {
	// get the current value
	value := gjson.Get(jsonData, mut.Path)
	if !value.Exists() {
		return "", fmt.Errorf("path %s did not match any element in the document", mut.Path)
	}

	tpl, err := template.New("mutation").Funcs(templateFuncMap()).Parse(mut.Template)
	if err != nil {
		return "", fmt.Errorf("failed to parse template %q: %w", mut.Template, err)
	}

	if ctx == nil {
		ctx = &TemplateMutationContext{}
	}
	ctx.Value = value

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, *ctx); err != nil {
		return "", fmt.Errorf("failed to execute template %q: %w", mut.Template, err)
	}

	replacement := strings.TrimSpace(buf.String())

	return sjson.Set(jsonData, mut.Path, replacement)
}
