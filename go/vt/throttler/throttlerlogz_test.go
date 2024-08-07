/*
Copyright 2019 The Vitess Authors.

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

package throttler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestThrottlerlogzHandler_MissingSlash(t *testing.T) {
	request, _ := http.NewRequest("GET", "/throttlerlogz", nil)
	response := httptest.NewRecorder()
	m := newManager()

	throttlerlogzHandler(response, request, m)

	got := response.Body.String()
	require.Contains(t, got, "invalid /throttlerlogz path", "/throttlerlogz without the slash does not work (the Go HTTP server does automatically redirect in practice though)")
}

func TestThrottlerlogzHandler_NonExistantThrottler(t *testing.T) {
	request, _ := http.NewRequest("GET", "/throttlerlogz/t1", nil)
	response := httptest.NewRecorder()

	throttlerlogzHandler(response, request, newManager())

	got := response.Body.String()
	require.Contains(t, got, "throttler not found", "/throttlerlogz page for non-existent t1 should not succeed")
}

func TestThrottlerlogzHandler(t *testing.T) {
	f := &managerTestFixture{}
	if err := f.setUp(); err != nil {
		t.Fatal(err)
	}
	defer f.tearDown()

	testcases := []struct {
		desc string
		r    Result
		want string
	}{
		{
			"increased rate",
			resultIncreased,
			`    <tr class="low">
      <td>00:00:01</td>
      <td>increased</td>
      <td>100</td>
      <td>100</td>
      <td>cell1-0000000101</td>
      <td>1s</td>
      <td>1.2s</td>
      <td>99</td>
      <td>good</td>
      <td></td>
      <td>95</td>
      <td>0</td>
      <td>I</td>
      <td>I</td>
      <td>I</td>
      <td>n/a</td>
      <td>n/a</td>
      <td>99</td>
      <td>0</td>
      <td>0</td>
      <td>0</td>
      <td>increased the rate</td>
    </tr>`,
		},
		{
			"decreased rate",
			resultDecreased,
			`    <tr class="medium">
      <td>00:00:05</td>
      <td>decreased</td>
      <td>200</td>
      <td>100</td>
      <td>cell1-0000000101</td>
      <td>2s</td>
      <td>3.8s</td>
      <td>200</td>
      <td>bad</td>
      <td></td>
      <td>95</td>
      <td>200</td>
      <td>I</td>
      <td>D</td>
      <td>D</td>
      <td>1s</td>
      <td>3.8s</td>
      <td>200</td>
      <td>150</td>
      <td>10</td>
      <td>20</td>
      <td>decreased the rate</td>
    </tr>`,
		},
		{
			"emergency state decreased the rate",
			resultEmergency,
			`    <tr class="high">
      <td>00:00:10</td>
      <td>decreased</td>
      <td>100</td>
      <td>50</td>
      <td>cell1-0000000101</td>
      <td>23s</td>
      <td>5.1s</td>
      <td>100</td>
      <td>bad</td>
      <td></td>
      <td>95</td>
      <td>100</td>
      <td>D</td>
      <td>E</td>
      <td>E</td>
      <td>2s</td>
      <td>5.1s</td>
      <td>0</td>
      <td>0</td>
      <td>0</td>
      <td>0</td>
      <td>emergency state decreased the rate</td>
    </tr>`,
		},
	}

	for _, tc := range testcases {
		request, _ := http.NewRequest("GET", "/throttlerlogz/t1", nil)
		response := httptest.NewRecorder()

		throttler, ok := f.t1.(*ThrottlerImpl)
		require.True(t, ok)
		throttler.maxReplicationLagModule.results.add(tc.r)
		throttlerlogzHandler(response, request, f.m)

		got := response.Body.String()
		require.Containsf(t, got, tc.want, "testcase '%v': result not shown in log", tc.desc)
	}
}
