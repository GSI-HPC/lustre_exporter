// -*- coding: utf-8 -*-
//
// © Copyright 2023-2025 GSI Helmholtzzentrum für Schwerionenforschung
//
// This software is distributed under
// the terms of the GNU General Public Licence version 3 (GPL Version 3),
// copied verbatim in the file "LICENCE".

package sources

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

var (
	lctlGetParamArgs                  = []string{"lctl", "get_param"}
	changelogTargetRegexPattern       = regexp.MustCompile(`mdd.([\w\d-]+-MDT[\d]+).changelog_users=`)
	changelogCurrentIndexRegexPattern = regexp.MustCompile(`current[_\s]index: (\d+)`)
	changelogUserRegexPattern         = regexp.MustCompile(`(?ms:(cl\d+[a-zA-Z-]*)\s+(\d+) \((\d+)\))`)
)

const (
	lctlParamChangelogUsers = "mdd.*-*.changelog_users"
)

type lustreLctlMetricCreator struct {
	lctlParam     string
	metricHandler func(string) ([]prometheus.Metric, error)
}

func init() {
	Factories["lctl"] = newLustreLctlSource
}

func regexCaptureChangelogTarget(textToMatch string) (string, error) {
	matchedTarget := changelogTargetRegexPattern.FindStringSubmatch(textToMatch)
	if matchedTarget != nil {
		if len(matchedTarget) == 2 {
			return matchedTarget[1], nil
		}
	}
	return "", fmt.Errorf("no target found in changelogs")
}

func regexCaptureChangelogCurrentIndex(textToMatch string) (float64, error) {
	matchedCurrentIndex := changelogCurrentIndexRegexPattern.FindStringSubmatch(textToMatch)
	if matchedCurrentIndex != nil {
		if len(matchedCurrentIndex) == 2 {
			currentIndex, err := strconv.ParseFloat(matchedCurrentIndex[1], 64)
			if err != nil {
				return -1, err
			}
			return currentIndex, nil
		}
	}
	return -1, fmt.Errorf("no current index found for changelogs")
}

func regexCaptureChangelogUser(textToMatch string) [][]string {
	return changelogUserRegexPattern.FindAllStringSubmatch(textToMatch, -1)

}

type lustreLctlSource struct {
	metricCreator []lustreLctlMetricCreator
}

func newLustreLctlSource() LustreSource {
	if LctlCommandMode {
		_, err := exec.LookPath("lctl")
		if err != nil {
			log.Error(err)
			return nil
		}
		_, err = exec.LookPath("sudo")
		if err != nil {
			log.Error(err)
			return nil
		}
	}
	var l lustreLctlSource
	l.metricCreator = []lustreLctlMetricCreator{}
	l.generateMDTMetricCreator(MdtEnabled)
	return &l
}

func (s *lustreLctlSource) Update(ch chan<- prometheus.Metric) (err error) {
	for _, metricCreator := range s.metricCreator {
		metricList, err := metricCreator.metricHandler(metricCreator.lctlParam)
		if err != nil {
			return fmt.Errorf("%s - %s", runtime.FuncForPC(reflect.ValueOf(metricCreator.metricHandler).Pointer()).Name(), err)
		}
		for _, metric := range metricList {
			ch <- metric
		}
	}
	return nil
}

func (s *lustreLctlSource) generateMDTMetricCreator(filter string) {
	if filter == extended {
		s.metricCreator = append(s.metricCreator,
			lustreLctlMetricCreator{
				lctlParam:     lctlParamChangelogUsers,
				metricHandler: s.createMDTChangelogUsersMetrics})
	}
}

func runLctlGetParam(lctlParam string) (string, error) {
	if LctlCommandMode {
		lctlCmdArgs := append(lctlGetParamArgs, lctlParam)
		if log.GetLevel() == log.DebugLevel {
			log.Debugf("Executing command: %s", "sudo "+strings.Join(lctlCmdArgs, " "))
		}
		out, err := exec.Command("sudo", lctlCmdArgs...).Output()
		if err != nil {
			return "", err
		}
		return string(out), nil
	}

	// for the testsuite:
	paramPath := strings.ReplaceAll(lctlParam, ".", OSPathSeparator)
	path := filepath.Join("lctl", paramPath)
	out, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (s *lustreLctlSource) createMDTChangelogUsersMetrics(lctlParam string) ([]prometheus.Metric, error) {
	metricList := make([]prometheus.Metric, 1)
	var target string
	var err error

	data, err := runLctlGetParam(lctlParam)
	if err != nil {
		return nil, err
	}

	target, err = regexCaptureChangelogTarget(data)
	if err != nil {
		return nil, err
	}

	currentIndex, err := regexCaptureChangelogCurrentIndex(data)
	if err != nil {
		return nil, err
	}

	metricList[0] = counterMetric(
		[]string{"component", "target"},
		[]string{"mdt", target},
		"changelog_current_index",
		"Changelog current index.",
		currentIndex)

	// Captures registered changelog user:
	for _, changelogUserFields := range regexCaptureChangelogUser(data) {

		id := changelogUserFields[1]

		index, err := strconv.ParseFloat(changelogUserFields[2], 64)
		if err != nil {
			return nil, err
		}

		idleSeconds, err := strconv.ParseFloat(changelogUserFields[3], 64)
		if err != nil {
			return nil, err
		}

		metric := counterMetric(
			[]string{"component", "target", "id"},
			[]string{"mdt", target, id},
			"changelog_user_index",
			"Index of registered changelog user.",
			index)
		metricList = append(metricList, metric)

		metric = gaugeMetric(
			[]string{"component", "target", "id"},
			[]string{"mdt", target, id},
			"changelog_user_idle_time",
			"Idle time in seconds of registered changelog user.",
			idleSeconds)
		metricList = append(metricList, metric)
	}

	return metricList, nil
}
