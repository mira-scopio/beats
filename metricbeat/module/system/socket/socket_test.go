// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

// +build linux

package socket

import (
	"net"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/elastic/beats/libbeat/common"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestData(t *testing.T) {
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	f := mbtest.NewReportingMetricSetV2(t, getConfig())
	err = mbtest.WriteEventsReporterV2(f, t, ".")
	if err != nil {
		t.Fatal("write", err)
	}
}

func TestFetch(t *testing.T) {
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	addr := ln.Addr().String()
	i := strings.LastIndex(addr, ":")
	listenerPort, err := strconv.Atoi(addr[i+1:])
	if err != nil {
		t.Fatal("failed to get port from addr", addr)
	}

	f := mbtest.NewReportingMetricSetV2(t, getConfig())
	events, errs := mbtest.ReportingFetchV2(f)

	assert.Empty(t, errs)
	if !assert.NotEmpty(t, events) {
		t.FailNow()
	}
	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(),
		events[0].BeatEvent("system", "socket").Fields.StringToPrint())

	var found bool
	for _, event := range events {
		s, err := event.BeatEvent("system", "socket").Fields.GetValue("system.socket")
		require.NoError(t, err)

		fields, ok := s.(common.MapStr)
		require.True(t, ok)

		port, ok := getRequiredValue(t, "local.port", fields).(int)
		if !ok {
			t.Fatal("local.port is not an int")
		}
		if port != listenerPort {
			continue
		}

		pid, ok := getRequiredValue(t, "process.pid", fields).(int)
		if !ok {
			t.Fatal("process.pid is not a int")
		}
		assert.Equal(t, os.Getpid(), pid)

		uid, ok := getRequiredValue(t, "user.id", fields).(uint32)
		if !ok {
			t.Fatal("user.id is not an uint32")
		}
		assert.EqualValues(t, os.Geteuid(), uid)

		dir, ok := getRequiredValue(t, "direction", fields).(string)
		if !ok {
			t.Fatal("direction is not a string")
		}
		assert.Equal(t, "listening", dir)

		_ = getRequiredValue(t, "process.cmdline", fields).(string)
		_ = getRequiredValue(t, "process.command", fields).(string)
		_ = getRequiredValue(t, "process.exe", fields).(string)

		found = true
		break
	}

	assert.True(t, found, "listener not found")
}

func getRequiredValue(t testing.TB, key string, m common.MapStr) interface{} {
	v, err := m.GetValue(key)
	if err != nil {
		t.Fatal(err)
	}
	if v == nil {
		t.Fatalf("key %v not found in %v", key, m)
	}
	return v
}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":     "system",
		"metricsets": []string{"socket"},
	}
}
