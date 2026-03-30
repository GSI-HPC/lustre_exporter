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
	"github.com/prometheus/client_golang/prometheus"
)

// ProcLocation is the source to pull proc files from. By default, use the '/proc' directory on the local node,
// but for testing purposes, specify 'proc' (without the leading '/') for the local files.
var ProcLocation = "/proc"

// SysLocation is the source to pull sys files from.
var SysLocation = "/sys"

// If LctlCommandMode is true it enables execution of lctl command which is meant to be executed on a Lustre client node.
// With false a local file is processed with test data.
var LctlCommandMode = true

// Namespace defines the namespace shared by all Lustre metrics.
const Namespace = "lustre"

// Factories contains the list of all sources.
var Factories = make(map[string]func() LustreSource)

// LustreSource is the interface that each source implements.
type LustreSource interface {
	Update(ch chan<- prometheus.Metric) (err error)
}

func counterMetric(labels []string, labelValues []string, name string, helpText string, value float64) prometheus.Metric {
	return prometheus.MustNewConstMetric(
		prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "", name),
			helpText,
			labels,
			nil,
		),
		prometheus.CounterValue,
		value,
		labelValues...,
	)
}

func gaugeMetric(labels []string, labelValues []string, name string, helpText string, value float64) prometheus.Metric {
	return prometheus.MustNewConstMetric(
		prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "", name),
			helpText,
			labels,
			nil,
		),
		prometheus.GaugeValue,
		value,
		labelValues...,
	)
}

//lint:ignore U1000 Ignore unused function for later use
func untypedMetric(labels []string, labelValues []string, name string, helpText string, value float64) prometheus.Metric {
	return prometheus.MustNewConstMetric(
		prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "", name),
			helpText,
			labels,
			nil,
		),
		prometheus.UntypedValue,
		value,
		labelValues...,
	)
}
