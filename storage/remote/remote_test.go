// Copyright 2021 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package remote

import (
	"context"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/storage"
	"github.com/prometheus/prometheus/util/testutil"
)

func TestRemoteStorageClient(t *testing.T) {
	server := httptest.NewServer(nil)
	defer server.Close()

	cfg := config.RemoteReadConfig{
		URL: &config.URL{URL: testutil.MustParseURL(server.URL)},
	}
	testClient, err := NewReadClient(1, &cfg)
	require.NoError(t, err)
}

func TestRemoteStorageClientWithRequiredMatchers(t *testing.T) {
	server := httptest.NewServer(nil)
	defer server.Close()

	// Test with different required matcher types
	configs := []config.RemoteReadConfig{
		{
			URL:              &config.URL{URL: testutil.MustParseURL(server.URL)},
			RequiredMatchers: []string{`{cluster="A"}`},
		},
		{
			URL:              &config.URL{URL: testutil.MustParseURL(server.URL)},
			RequiredMatchers: []string{`{cluster=~"A.*"}`},
		},
		{
			URL:              &config.URL{URL: testutil.MustParseURL(server.URL)},
			RequiredMatchers: []string{`{cluster!="B"}`},
		},
		{
			URL:              &config.URL{URL: testutil.MustParseURL(server.URL)},
			RequiredMatchers: []string{`{cluster!~"B.*"}`},
		},
		{
			URL:              &config.URL{URL: testutil.MustParseURL(server.URL)},
			RequiredMatchers: []string{`{cluster="A", environment="prod"}`},
		},
	}

	for i, cfg := range configs {
		t.Run(fmt.Sprintf("Config%d", i), func(t *testing.T) {
			client, err := NewReadClient(i, &cfg)
			require.NoError(t, err)

			// Verify the client has the required matchers
			c, ok := client.(*Client)
			require.True(t, ok)
			require.NotEmpty(t, c.requiredMatchers)
		})
	}
}

func TestRequiredMatchers(t *testing.T) {
	server := httptest.NewServer(nil)
	defer server.Close()

	tests := []struct {
		name                string
		requiredMatchers    []string
		queryMatchers       []*labels.Matcher
		expectNoopSeriesSet bool
	}{
		{
			name:             "exact match",
			requiredMatchers: []string{`{cluster="A"}`},
			queryMatchers: []*labels.Matcher{
				labels.MustNewMatcher(labels.MatchEqual, "cluster", "A"),
			},
			expectNoopSeriesSet: false,
		},
		{
			name:             "no match",
			requiredMatchers: []string{`{cluster="A"}`},
			queryMatchers: []*labels.Matcher{
				labels.MustNewMatcher(labels.MatchEqual, "cluster", "B"),
			},
			expectNoopSeriesSet: true,
		},
		{
			name:             "regex match",
			requiredMatchers: []string{`{cluster=~"A.*"}`},
			queryMatchers: []*labels.Matcher{
				labels.MustNewMatcher(labels.MatchEqual, "cluster", "ABC"),
			},
			expectNoopSeriesSet: false,
		},
		{
			name:             "not equal match",
			requiredMatchers: []string{`{cluster!="B"}`},
			queryMatchers: []*labels.Matcher{
				labels.MustNewMatcher(labels.MatchEqual, "cluster", "A"),
			},
			expectNoopSeriesSet: false,
		},
		{
			name:             "not regex match",
			requiredMatchers: []string{`{cluster!~"B.*"}`},
			queryMatchers: []*labels.Matcher{
				labels.MustNewMatcher(labels.MatchEqual, "cluster", "A"),
			},
			expectNoopSeriesSet: false,
		},
		{
			name:             "multiple matchers - all match",
			requiredMatchers: []string{`{cluster="A", environment="prod"}`},
			queryMatchers: []*labels.Matcher{
				labels.MustNewMatcher(labels.MatchEqual, "cluster", "A"),
				labels.MustNewMatcher(labels.MatchEqual, "environment", "prod"),
			},
			expectNoopSeriesSet: false,
		},
		{
			name:             "multiple matchers - partial match",
			requiredMatchers: []string{`{cluster="A", environment="prod"}`},
			queryMatchers: []*labels.Matcher{
				labels.MustNewMatcher(labels.MatchEqual, "cluster", "A"),
				labels.MustNewMatcher(labels.MatchEqual, "environment", "dev"),
			},
			expectNoopSeriesSet: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.RemoteReadConfig{
				URL:              &config.URL{URL: testutil.MustParseURL(server.URL)},
				RequiredMatchers: tc.requiredMatchers,
			}
			client, err := NewReadClient(1, &cfg)
			require.NoError(t, err)

			queryable := NewSampleAndChunkQueryableClient(client, labels.Labels{}, false, func() (int64, error) { return 0, nil })
			querier, err := queryable.Querier(0, 10)
			require.NoError(t, err)

			result := querier.Select(context.Background(), true, nil, tc.queryMatchers...)

			if tc.expectNoopSeriesSet {
				// Check if it's a NoopSeriesSet
				_, ok := result.(*storage.NoopSeriesSet)
				require.True(t, ok, "Expected NoopSeriesSet but got different type")
			} else {
				// Should not be a NoopSeriesSet
				_, ok := result.(*storage.NoopSeriesSet)
				require.False(t, ok, "Expected not to get NoopSeriesSet but got one")
			}
		})
	}
}
