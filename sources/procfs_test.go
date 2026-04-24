// (C) Copyright 2017 Hewlett Packard Enterprise Development LP
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

package sources

import (
	"testing"
)

func TestGetJobNum(t *testing.T) {

	tests := map[string]string{
		"job_id: 1234":                      "1234",
		"job_id: ABCD":                      "ABCD",
		"job_id:  abc .0123 .-_+ AB.1000  ": "abc .0123 .-_+ AB.1000",
		"job_id:            kworker/86:1.0": "kworker/86:1.0",
		"job_id: (ostnamed.0)":              "(ostnamed.0)",
		"job_id: \"1125\"":                  "1125",
		"job_id: \"kworker\\XA\\1125\")":    "kworker\\XA\\1125",
	}

	for testString, expected := range tests {
		jobID, err := getJobNum(testString)
		if err != nil {
			t.Fatal(err)
		}
		if jobID != expected {
			t.Fatalf("Received an unexpected jobid. Expected: %s, Got: %s", expected, jobID)
		}
	}

	_, err := getJobNum("")
	if err == nil {
		t.Fatal("An error was expected for an empty jobid, but not received")
	}

	testJobBlock := `- job_id:
  snapshot_time:   1493326943
  read_bytes:      { samples:         126, unit: bytes, min: 1048576, max: 1048576, sum:       132120576 }
  write_bytes:     { samples:         262, unit: bytes, min: 1048576, max: 1048576, sum:       274726912 }
  getattr:         { samples:           1, unit:  reqs }
  setattr:         { samples:           2, unit:  reqs }
  punch:           { samples:           3, unit:  reqs }
  sync:            { samples:           4, unit:  reqs }
  destroy:         { samples:           5, unit:  reqs }
  create:          { samples:           6, unit:  reqs }
  statfs:          { samples:           7, unit:  reqs }
  get_info:        { samples:           8, unit:  reqs }
  set_info:        { samples:           9, unit:  reqs }
  quotactl:        { samples:           10, unit:  reqs }
  - job_id: 28
  snapshot_time:   1493326943
  read_bytes:      { samples:         126, unit: bytes, min: 1048576, max: 1048576, sum:       132120576 }
  write_bytes:     { samples:         262, unit: bytes, min: 1048576, max: 1048576, sum:       274726912 }
  getattr:         { samples:           1, unit:  reqs }
  setattr:         { samples:           2, unit:  reqs }
  punch:           { samples:           3, unit:  reqs }
  sync:            { samples:           4, unit:  reqs }
  destroy:         { samples:           5, unit:  reqs }
  create:          { samples:           6, unit:  reqs }
  statfs:          { samples:           7, unit:  reqs }
  get_info:        { samples:           8, unit:  reqs }
  set_info:        { samples:           9, unit:  reqs }
  quotactl:        { samples:           10, unit:  reqs }`
	expectedJobId := ""

	jobID, err := getJobNum(testJobBlock)
	if err == nil {
		t.Fatal("An error was expected for an empty jobid, but not received")
	}
	if jobID != expectedJobId {
		t.Fatalf("Expected empty jobid. Got: %s", jobID)
	}
}

func TestGetJobStats(t *testing.T) {
	testJobBlock := `- job_id:          29
  snapshot_time:   1493326943
  read_bytes:      { samples:         126, unit: bytes, min: 1048576, max: 1048576, sum:       132120576 }
  write_bytes:     { samples:         262, unit: bytes, min: 1048576, max: 1048576, sum:       274726912 }
  getattr:         { samples:           1, unit:  reqs }
  setattr:         { samples:           2, unit:  reqs }
  punch:           { samples:           3, unit:  reqs }
  sync:            { samples:           4, unit:  reqs }
  destroy:         { samples:           5, unit:  reqs }
  create:          { samples:           6, unit:  reqs }
  statfs:          { samples:           7, unit:  reqs }
  get_info:        { samples:           8, unit:  reqs }
  set_info:        { samples:           9, unit:  reqs }
  quotactl:        { samples:           10, unit:  reqs }`
	testJobID := "29"

	testPromName := "job_read_bytes_total"
	testHelpText := readTotalHelp
	expected := float64(132120576)

	metricList, err := getJobStatsIOMetrics(testJobBlock, testJobID, testPromName, testHelpText)
	if err != nil {
		t.Fatal(err)
	}
	if l := len(metricList); l != 1 {
		t.Fatalf("Retrieved an unexpected number of items. Expected: %d, Got: %d", 1, l)
	}
	if metricList[0].value != expected {
		t.Fatalf("Retrieved an unexpected value. Expected: %f, Got: %f", expected, metricList[0].value)
	}
	if metricList[0].help != readTotalHelp {
		t.Fatal("Retrieved an unexpected help text.")
	}
	if metricList[0].title != testPromName {
		t.Fatalf("Retrieved an unexpected name. Expected: %s, Got: %s", testPromName, metricList[0].title)
	}

	testPromName = "job_write_bytes_total"
	testHelpText = writeTotalHelp
	expected = float64(274726912)

	metricList, err = getJobStatsIOMetrics(testJobBlock, testJobID, testPromName, testHelpText)
	if err != nil {
		t.Fatal(err)
	}
	if l := len(metricList); l != 1 {
		t.Fatalf("Retrieved an unexpected number of items. Expected: %d, Got: %d", 1, l)
	}
	if metricList[0].value != expected {
		t.Fatalf("Retrieved an unexpected value. Expected: %f, Got: %f", expected, metricList[0].value)
	}
	if metricList[0].help != writeTotalHelp {
		t.Fatal("Retrieved an unexpected help text.")
	}
	if metricList[0].title != testPromName {
		t.Fatalf("Retrieved an unexpected name. Expected: %s, Got: %s", testPromName, metricList[0].title)
	}

	testPromName = "job_stats_total"
	testHelpText = jobStatsHelp

	metricList, err = getJobStatsOperationMetrics(testJobBlock, testJobID, testPromName, testHelpText)
	if err != nil {
		t.Fatal(err)
	}
	if l := len(metricList); l != 10 {
		t.Fatalf("Retrieved an unexpected number of items. Expected: %d, Got: %d", 1, l)
	}
	if metricList[9].value != float64(10) {
		t.Fatalf("Retrieved an unexpected value. Expected: %f, Got: %f", float64(10), metricList[9].value)
	}
	if metricList[3].help != jobStatsHelp {
		t.Fatal("Retrieved an unexpected help text.")
	}
	if metricList[6].title != testPromName {
		t.Fatalf("Retrieved an unexpected name. Expected: %s, Got: %s", testPromName, metricList[6].title)
	}

	testPromName = "dne"
	testHelpText = "Help for DNE"

	metricList, err = getJobStatsIOMetrics(testJobBlock, testJobID, testPromName, testHelpText)
	if err != nil {
		t.Fatal(err)
	}
	if l := len(metricList); l != 0 {
		t.Fatalf("Retrieved an unexpected number of items. Expected: %d, Got: %d", 0, l)
	}

	testJobBlock = `- job_id:          30
  snapshot_time:   1493326960
  read_bytes:      { samples:           0, unit: bytes, min: 0, max: 0, sum: 0 }
  write_bytes:     { samples:           0, unit: bytes, min: 0, max: 0, sum: 0 }
  getattr:         { samples:           1, unit:  reqs }
  setattr:         { samples:           2, unit:  reqs }
  punch:           { samples:           3, unit:  reqs }
  sync:            { samples:           4, unit:  reqs }
  destroy:         { samples:           5, unit:  reqs }
  create:          { samples:           0, unit:  reqs }
  statfs:          { samples:           0, unit:  reqs }
  get_info:        { samples:           0, unit:  reqs }
  set_info:        { samples:           0, unit:  reqs }
  quotactl:        { samples:           6, unit:  reqs }`
	testJobID = "30"

	testPromName = "job_read_bytes_total"
	testHelpText = readTotalHelp

	metricList, err = getJobStatsIOMetrics(testJobBlock, testJobID, testPromName, testHelpText)
	if err != nil {
		t.Fatal(err)
	}
	if metricList != nil {
		t.Fatal("Retrieved metric object. Expected nil, since 0 values should be dropped.")
	}

	testPromName = "job_write_bytes_total"
	testHelpText = writeTotalHelp

	metricList, err = getJobStatsIOMetrics(testJobBlock, testJobID, testPromName, testHelpText)
	if err != nil {
		t.Fatal(err)
	}
	if metricList != nil {
		t.Fatal("Retrieved metric object. Expected nil, since 0 values should be dropped.")
	}

	testPromName = "job_stats_total"
	testHelpText = jobStatsHelp

	metricList, err = getJobStatsOperationMetrics(testJobBlock, testJobID, testPromName, testHelpText)
	if err != nil {
		t.Fatal(err)
	}
	if l := len(metricList); l != 6 {
		t.Fatalf("Retrieved an unexpected number of items. Expected: %d, Got: %d", 6, l)
	}
	if metricList[5].value != float64(6) {
		t.Fatalf("Retrieved an unexpected value. Expected: %f, Got: %f", float64(6), metricList[5].value)
	}
	if metricList[0].help != jobStatsHelp {
		t.Fatal("Retrieved an unexpected help text.")
	}
	if metricList[5].title != testPromName {
		t.Fatalf("Retrieved an unexpected name. Expected: %s, Got: %s", testPromName, metricList[5].title)
	}
}

func TestJobStatBlockIds(t *testing.T) {

	testJobBlocks := `- job_id: 67
	snapshot_time:   1493326943
	read_bytes:      { samples:         126, unit: bytes, min: 1048576, max: 1048576, sum:       132120576 }
	write_bytes:     { samples:         262, unit: bytes, min: 1048576, max: 1048576, sum:       274726912 }
	getattr:         { samples:           1, unit:  reqs }
	setattr:         { samples:           2, unit:  reqs }
	punch:           { samples:           3, unit:  reqs }
	sync:            { samples:           4, unit:  reqs }
	destroy:         { samples:           5, unit:  reqs }
	create:          { samples:           6, unit:  reqs }
	statfs:          { samples:           7, unit:  reqs }
	get_info:        { samples:           8, unit:  reqs }
	set_info:        { samples:           9, unit:  reqs }
	quotactl:        { samples:           10, unit:  reqs }
	- job_id: 28
	snapshot_time:   1493326943
	read_bytes:      { samples:         126, unit: bytes, min: 1048576, max: 1048576, sum:       132120576 }
	write_bytes:     { samples:         262, unit: bytes, min: 1048576, max: 1048576, sum:       274726912 }
	getattr:         { samples:           1, unit:  reqs }
	setattr:         { samples:           2, unit:  reqs }
	punch:           { samples:           3, unit:  reqs }
	sync:            { samples:           4, unit:  reqs }
	destroy:         { samples:           5, unit:  reqs }
	create:          { samples:           6, unit:  reqs }
	statfs:          { samples:           7, unit:  reqs }
	get_info:        { samples:           8, unit:  reqs }
	set_info:        { samples:           9, unit:  reqs }
	quotactl:        { samples:           10, unit:  reqs }`

	expectedJobIds := []string{"67", "28"}

	matchedJobBlocks := regexCaptureJobStats(testJobBlocks)

	if l := len(matchedJobBlocks); l != 2 {
		t.Fatalf("Retrieved an unexpected number of items. Expected: %d, Got: %d", 2, l)
	}

	for index, expected := range expectedJobIds {
		jobId, err := getJobNum(matchedJobBlocks[index])
		if err != nil {
			t.Fatal(err)
		}
		if jobId != expected {
			t.Fatalf("Received an unexpected jobId. Expected: %s, Got: %s", expected, jobId)
		}
	}
}
