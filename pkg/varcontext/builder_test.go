// Copyright 2018 the Service Broker Project Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package varcontext

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestContextBuilder(t *testing.T) {
	cases := map[string]struct {
		Builder     *ContextBuilder
		Expected    map[string]interface{}
		ErrContains string
	}{
		"an empty context": {
			Builder:     Builder(),
			Expected:    map[string]interface{}{},
			ErrContains: "",
		},

		// MergeMap
		"MergeMap blank okay": {
			Builder:  Builder().MergeMap(map[string]interface{}{}),
			Expected: map[string]interface{}{},
		},
		"MergeMap multi-key": {
			Builder:  Builder().MergeMap(map[string]interface{}{"a": "a", "b": "b"}),
			Expected: map[string]interface{}{"a": "a", "b": "b"},
		},
		"MergeMap overwrite": {
			Builder:  Builder().MergeMap(map[string]interface{}{"a": "a"}).MergeMap(map[string]interface{}{"a": "aaa"}),
			Expected: map[string]interface{}{"a": "aaa"},
		},

		// MergeDefaults
		"MergeDefaults no defaults": {
			Builder:  Builder().MergeDefaults([]DefaultVariable{{Name: "foo"}}),
			Expected: map[string]interface{}{},
		},
		"MergeDefaults non-string": {
			Builder:  Builder().MergeDefaults([]DefaultVariable{{Name: "h2g2", Default: 42}}),
			Expected: map[string]interface{}{"h2g2": 42},
		},
		"MergeDefaults basic-string": {
			Builder:  Builder().MergeDefaults([]DefaultVariable{{Name: "a", Default: "no-template"}}),
			Expected: map[string]interface{}{"a": "no-template"},
		},
		"MergeDefaults template string": {
			Builder:  Builder().MergeDefaults([]DefaultVariable{{Name: "a", Default: "a"}, {Name: "b", Default: "${a}"}}),
			Expected: map[string]interface{}{"a": "a", "b": "a"},
		},
		"MergeDefaults no-overwrite": {
			Builder:  Builder().MergeDefaults([]DefaultVariable{{Name: "a", Default: "a"}, {Name: "a", Default: "b", Overwrite: false}}),
			Expected: map[string]interface{}{"a": "a"},
		},
		"MergeDefaults overwrite": {
			Builder:  Builder().MergeDefaults([]DefaultVariable{{Name: "a", Default: "a"}, {Name: "a", Default: "b", Overwrite: true}}),
			Expected: map[string]interface{}{"a": "b"},
		},

		// MergeEvalResult
		"MergeEvalResult accumulates context": {
			Builder:  Builder().MergeEvalResult("a", "a").MergeEvalResult("b", "${a}"),
			Expected: map[string]interface{}{"a": "a", "b": "a"},
		},
		"MergeEvalResult errors": {
			Builder:     Builder().MergeEvalResult("a", "${dne}"),
			ErrContains: `couldn't compute the value for "a"`,
		},

		// MergeJsonObject
		"MergeJsonObject blank message": {
			Builder:  Builder().MergeJsonObject(json.RawMessage{}),
			Expected: map[string]interface{}{},
		},
		"MergeJsonObject valid message": {
			Builder:  Builder().MergeJsonObject(json.RawMessage(`{"a":"a"}`)),
			Expected: map[string]interface{}{"a": "a"},
		},
		"MergeJsonObject invalid message": {
			Builder:     Builder().MergeJsonObject(json.RawMessage(`{{{}}}`)),
			ErrContains: "invalid character '{'",
		},

		// MergeStruct
		"MergeStruct without JSON Tags": {
			Builder:  Builder().MergeStruct(struct{ Name string }{Name: "Foo"}),
			Expected: map[string]interface{}{"Name": "Foo"},
		},
		"MergeStruct with JSON Tags": {
			Builder: Builder().MergeStruct(struct {
				Name string `json:"username"`
			}{Name: "Foo"}),
			Expected: map[string]interface{}{"username": "Foo"},
		},

		// constants
		"Basic constants": {
			Builder: Builder().
				SetEvalConstants(map[string]interface{}{"PI": 3.14}).
				MergeEvalResult("out", "${PI}"),
			Expected: map[string]interface{}{"out": "3.14"},
		},
		"User overrides constant": {
			Builder: Builder().
				SetEvalConstants(map[string]interface{}{"PI": 3.14}).
				MergeMap(map[string]interface{}{"PI": 3.2}). // reassign incorrectly, https://en.wikipedia.org/wiki/Indiana_Pi_Bill
				MergeEvalResult("PI", "${PI}"),              // test which PI gets referenced
			Expected: map[string]interface{}{"PI": "3.14"},
		},
	}

	for tn, tc := range cases {
		t.Run(tn, func(t *testing.T) {

			vc, err := tc.Builder.Build()

			if vc == nil && tc.Expected != nil {
				t.Fatalf("Expected: %v, got: %v", tc.Expected, vc)
			}

			if vc != nil && !reflect.DeepEqual(vc.ToMap(), tc.Expected) {
				t.Errorf("Expected: %v, got: %v", tc.Expected, vc.ToMap())
			}

			switch {
			case err == nil && tc.ErrContains == "":
				break
			case err == nil && tc.ErrContains != "":
				t.Errorf("Got no error when %q was expected", tc.ErrContains)
			case err != nil && tc.ErrContains == "":
				t.Errorf("Got error %v when none was expected", err)
			case !strings.Contains(err.Error(), tc.ErrContains):
				t.Errorf("Got error %v, but expected it to contain %q", err, tc.ErrContains)
			}
		})
	}
}

func ExampleContextBuilder_BuildMap() {
	_, e := Builder().MergeEvalResult("a", "${assert(false, \"failure!\")}").BuildMap()
	fmt.Printf("Error: %v\n", e)

	m, _ := Builder().MergeEvalResult("a", "${1+1}").BuildMap()
	fmt.Printf("Map: %v\n", m)

	//Output: Error: 1 error(s) occurred: couldn't compute the value for "a", template: "${assert(false, \"failure!\")}", assert: Assertion failed: failure!
	// Map: map[a:2]
}
