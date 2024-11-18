/*
Copyright 2020 The Kubernetes Authors.
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

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetOrDefault(t *testing.T) {
	tests := []struct {
		name       string
		arr        []string
		index      int
		defaultVal string
		expected   string
	}{
		{
			name:       "index middle of array",
			arr:        []string{"a", "b", "c"},
			index:      1,
			defaultVal: "default",
			expected:   "b",
		},
		{
			name:       "index start of array",
			arr:        []string{"a", "b", "c"},
			index:      0,
			defaultVal: "default",
			expected:   "a",
		},
		{
			name:       "index end of array",
			arr:        []string{"a", "b", "c"},
			index:      2,
			defaultVal: "default",
			expected:   "c",
		},
		{
			name:       "index out of array",
			arr:        []string{"a", "b", "c"},
			index:      3,
			defaultVal: "default",
			expected:   "default",
		},
		{
			name:       "index less than zero",
			arr:        []string{"a", "b", "c"},
			index:      -3,
			defaultVal: "default",
			expected:   "default",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := getOrDefault(test.arr, test.index, test.defaultVal)
			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestStringToStringConv(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]string
	}{
		{
			name:     "single element map",
			input:    "key=value",
			expected: map[string]string{"key": "value"},
		},
		{
			name:     "empty map",
			input:    "",
			expected: map[string]string{},
		},
		{
			name:     "two element map",
			input:    "key=value,key2=value2",
			expected: map[string]string{"key": "value", "key2": "value2"},
		},
		{
			name:     "3 element map",
			input:    "key=value,key2=value2,key3=value3",
			expected: map[string]string{"key": "value", "key2": "value2", "key3": "value3"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := stringToStringConv(test.input)
			assert.NoError(t, err)
			assert.Equal(t, test.expected, actual)
		})
	}
}
