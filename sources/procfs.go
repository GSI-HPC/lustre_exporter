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
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

const (
	// Help text dedicated to the 'stats' files
	readSamplesHelp  string = "Total number of reads that have been recorded."
	readMaximumHelp  string = "The maximum read size in bytes."
	readMinimumHelp  string = "The minimum read size in bytes."
	readTotalHelp    string = "The total number of bytes that have been read."
	writeSamplesHelp string = "Total number of writes that have been recorded."
	writeMaximumHelp string = "The maximum write size in bytes."
	writeMinimumHelp string = "The minimum write size in bytes."
	writeTotalHelp   string = "The total number of bytes that have been written."
	jobStatsHelp     string = "Number of operations the filesystem has performed."
	statsHelp        string = "Number of operations the filesystem has performed."

	// Help text dedicated to the 'brw_stats' file
	pagesPerBlockRWHelp    string = "Total number of pages per block RPC."
	discontiguousPagesHelp string = "Total number of logical discontinuities per RPC."
	ioTimeHelp             string = "Total time in milliseconds the filesystem has spent processing various object sizes."
	diskIOSizeHelp         string = "Total number of operations the filesystem has performed for the given size."
	diskIOsInFlightHelp    string = "Current number of I/O operations that are processing during the snapshot."

	// Help text dedicated to the 'rpc_stats' file
	pagesPerRPCHelp  string = "Total number of pages per RPC."
	rpcsInFlightHelp string = "Current number of RPCs that are processing during the snapshot."
	offsetHelp       string = "Current RPC offset by size."

	// Help text dedicated to the 'encrypt_page_pools' file
	physicalPagesHelp     string = "Capacity of physical memory."
	pagesPerPoolHelp      string = "Number of pages per pool."
	maxPagesHelp          string = "Maximum number of pages that can be held."
	maxPoolsHelp          string = "Number of pools."
	totalPagesHelp        string = "Number of pages in all pools."
	totalFreeHelp         string = "Current number of pages available."
	maxPagesReachedHelp   string = "Total number of pages reached."
	growsHelp             string = "Total number of grows."
	growsFailureHelp      string = "Total number of failures while attempting to add pages."
	shrinksHelp           string = "Total number of shrinks."
	cacheAccessHelp       string = "Total number of times cache has been accessed."
	cacheMissingHelp      string = "Total number of cache misses."
	lowFreeMarkHelp       string = "Lowest number of free pages reached."
	maxWaitQueueDepthHelp string = "Maximum waitqueue length."
	outOfMemHelp          string = "Total number of out of memory requests."

	//repeated strings replaced by constants
	mdStats          string = "md_stats"
	encryptPagePools string = "encrypt_page_pools"
)

var (
	// OstEnabled specifies whether to collect OST metrics
	OstEnabled string
	// MdtEnabled specifies whether to collect MDT metrics
	MdtEnabled string
	// MgsEnabled specifies whether to collect MGS metrics
	MgsEnabled string
	// MdsEnabled specifies whether to collect MDS metrics
	MdsEnabled string
	// ClientEnabled specifies whether to collect Client metrics
	ClientEnabled string
	// GenericEnabled specifies whether to collect Generic metrics
	GenericEnabled string
)

type lustreJobsMetric struct {
	jobID string
	lustreStatsMetric
}

type lustreBRWMetric struct {
	size      string
	operation string
	value     string
}

type multistatParsingStruct struct {
	index   int
	pattern string
}

func init() {
	Factories["procfs"] = newLustreProcFsSource
}

type lustreProcFsSource struct {
	lustreProcMetrics []lustreProcMetric
	basePath          string
}

func (s *lustreProcFsSource) generateOSTMetricTemplates(filter string) {
	metricMap := map[string][]lustreHelpStruct{
		"obdfilter/*-OST*": {
			{"brw_size", "brw_size_megabytes", "Block read/write size in megabytes", gaugeMetric, false, extended},
			{"grant_compat_disable", "grant_compat_disabled", "Binary indicator as to whether clients with OBD_CONNECT_GRANT_PARAM setting will be granted space", gaugeMetric, false, extended},
			{"job_cleanup_interval", "job_cleanup_interval_seconds", "Interval in seconds between cleanup of tuning statistics", gaugeMetric, false, extended},
			{"job_stats", "job_read_samples_total", readSamplesHelp, counterMetric, false, core},
			{"job_stats", "job_read_minimum_size_bytes", readMinimumHelp, gaugeMetric, false, core},
			{"job_stats", "job_read_maximum_size_bytes", readMaximumHelp, gaugeMetric, false, core},
			{"job_stats", "job_read_bytes_total", readTotalHelp, counterMetric, false, core},
			{"job_stats", "job_write_samples_total", writeSamplesHelp, counterMetric, false, core},
			{"job_stats", "job_write_minimum_size_bytes", writeMinimumHelp, gaugeMetric, false, extended},
			{"job_stats", "job_write_maximum_size_bytes", writeMaximumHelp, gaugeMetric, false, extended},
			{"job_stats", "job_write_bytes_total", writeTotalHelp, counterMetric, false, core},
			{"job_stats", "job_stats_total", jobStatsHelp, counterMetric, true, core},
			{"num_exports", "exports_total", "Total number of times the pool has been exported", counterMetric, false, core},
			{"recovery_time_hard", "recovery_time_hard_seconds", "Maximum timeout 'recover_time_soft' can increment to for a single server", gaugeMetric, false, extended},
			{"recovery_time_soft", "recovery_time_soft_seconds", "Duration in seconds for a client to attempt to reconnect after a crash (automatically incremented if servers are still in an error state)", gaugeMetric, false, extended},
			{"stats", "read_samples_total", readSamplesHelp, counterMetric, false, core},
			{"stats", "read_minimum_size_bytes", readMinimumHelp, gaugeMetric, false, extended},
			{"stats", "read_maximum_size_bytes", readMaximumHelp, gaugeMetric, false, extended},
			{"stats", "read_bytes_total", readTotalHelp, counterMetric, false, core},
			{"stats", "write_samples_total", writeSamplesHelp, counterMetric, false, core},
			{"stats", "write_minimum_size_bytes", writeMinimumHelp, gaugeMetric, false, extended},
			{"stats", "write_maximum_size_bytes", writeMaximumHelp, gaugeMetric, false, extended},
			{"stats", "write_bytes_total", writeTotalHelp, counterMetric, false, core},
			{"stats", "stats_total", statsHelp, counterMetric, true, core},
			{"tot_dirty", "exports_dirty_total", "Total number of exports that have been marked dirty", counterMetric, false, core},
			{"tot_granted", "exports_granted_total", "Total number of exports that have been marked granted", counterMetric, false, core},
			{"tot_pending", "exports_pending_total", "Total number of exports that have been marked pending", counterMetric, false, core},
		},
		"osd-*/*-OST*": {
			{"blocksize", "blocksize_bytes", "Filesystem block size in bytes", gaugeMetric, false, core},
			{"brw_stats", "pages_per_bulk_rw_total", pagesPerBlockRWHelp, counterMetric, false, extended},
			{"brw_stats", "discontiguous_pages_total", discontiguousPagesHelp, counterMetric, false, extended},
			{"brw_stats", "disk_io", diskIOsInFlightHelp, gaugeMetric, false, core},
			{"brw_stats", "io_time_milliseconds_total", ioTimeHelp, counterMetric, false, core},
			{"brw_stats", "disk_io_total", diskIOSizeHelp, counterMetric, false, core},
			{"filesfree", "inodes_free", "The number of inodes (objects) available", gaugeMetric, false, core},
			{"filestotal", "inodes_maximum", "The maximum number of inodes (objects) the filesystem can hold", gaugeMetric, false, core},
			{"kbytesfree", "free_kibibytes", "Number of kibibytes free in the pool", gaugeMetric, false, core},
			{"kbytesavail", "available_kibibytes", "Number of kibibytes readily available in the pool", gaugeMetric, false, core},
			{"kbytestotal", "capacity_kibibytes", "Capacity of the pool in kibibytes", gaugeMetric, false, core},
		},
	}
	for path := range metricMap {
		for _, item := range metricMap[path] {
			if filter == extended || item.priorityLevel == core {
				newMetric := newLustreProcMetric(&item, "ost", path)
				s.lustreProcMetrics = append(s.lustreProcMetrics, *newMetric)
			}
		}
	}
}

func (s *lustreProcFsSource) generateMDTMetricTemplates(filter string) {
	metricMap := map[string][]lustreHelpStruct{
		"osd-*/*-MDT*": {
			{"blocksize", "blocksize_bytes", "Filesystem block size in bytes", gaugeMetric, false, core},
			{"filesfree", "inodes_free", "The number of inodes (objects) available", gaugeMetric, false, core},
			{"filestotal", "inodes_maximum", "The maximum number of inodes (objects) the filesystem can hold", gaugeMetric, false, core},
			{"kbytesavail", "available_kibibytes", "Number of kibibytes readily available in the pool", gaugeMetric, false, core},
			{"kbytesfree", "free_kibibytes", "Number of kibibytes free in the pool", gaugeMetric, false, core},
			{"kbytestotal", "capacity_kibibytes", "Capacity of the pool in kibibytes", gaugeMetric, false, core},
		},
		"mdt/*": {
			{mdStats, "stats_total", statsHelp, counterMetric, true, core},
			{"num_exports", "exports_total", "Total number of times the pool has been exported", counterMetric, false, core},
			{"job_stats", "job_stats_total", jobStatsHelp, counterMetric, true, core},
		},
	}
	for path := range metricMap {
		for _, item := range metricMap[path] {
			if filter == extended || item.priorityLevel == core {
				newMetric := newLustreProcMetric(&item, "mdt", path)
				s.lustreProcMetrics = append(s.lustreProcMetrics, *newMetric)
			}
		}
	}
}

func (s *lustreProcFsSource) generateMGSMetricTemplates(filter string) {
	metricMap := map[string][]lustreHelpStruct{
		"mgs/MGS/osd/": {
			{"blocksize", "blocksize_bytes", "Filesystem block size in bytes", gaugeMetric, false, core},
			{"filesfree", "inodes_free", "The number of inodes (objects) available", gaugeMetric, false, core},
			{"filestotal", "inodes_maximum", "The maximum number of inodes (objects) the filesystem can hold", gaugeMetric, false, core},
			{"kbytesavail", "available_kibibytes", "Number of kibibytes readily available in the pool", gaugeMetric, false, core},
			{"kbytesfree", "free_kibibytes", "Number of kibibytes free in the pool", gaugeMetric, false, core},
			{"kbytestotal", "capacity_kibibytes", "Capacity of the pool in kibibytes", gaugeMetric, false, core},
		},
	}
	for path := range metricMap {
		for _, item := range metricMap[path] {
			if filter == extended || item.priorityLevel == core {
				newMetric := newLustreProcMetric(&item, "mgs", path)
				s.lustreProcMetrics = append(s.lustreProcMetrics, *newMetric)
			}
		}
	}
}

func (s *lustreProcFsSource) generateMDSMetricTemplates(filter string) {
	metricMap := map[string][]lustreHelpStruct{}
	for path := range metricMap {
		for _, item := range metricMap[path] {
			if filter == extended || item.priorityLevel == core {
				newMetric := newLustreProcMetric(&item, "mds", path)
				s.lustreProcMetrics = append(s.lustreProcMetrics, *newMetric)
			}
		}
	}
}

func (s *lustreProcFsSource) generateClientMetricTemplates(filter string) {
	metricMap := map[string][]lustreHelpStruct{
		"llite/*": {
			{"blocksize", "blocksize_bytes", "Filesystem block size in bytes", gaugeMetric, false, core},
			{"checksum_pages", "checksum_pages_enabled", "Returns '1' if data checksumming is enabled for the client", gaugeMetric, false, extended},
			{"default_easize", "default_ea_size_bytes", "Default Extended Attribute (EA) size in bytes", gaugeMetric, false, extended},
			{"filesfree", "inodes_free", "The number of inodes (objects) available", gaugeMetric, false, core},
			{"filestotal", "inodes_maximum", "The maximum number of inodes (objects) the filesystem can hold", gaugeMetric, false, core},
			{"kbytesavail", "available_kibibytes", "Number of kibibytes readily available in the pool", gaugeMetric, false, core},
			{"kbytesfree", "free_kibibytes", "Number of kibibytes free in the pool", gaugeMetric, false, core},
			{"kbytestotal", "capacity_kibibytes", "Capacity of the pool in kibibytes", gaugeMetric, false, core},
			{"lazystatfs", "lazystatfs_enabled", "Returns '1' if lazystatfs (a non-blocking alternative to statfs) is enabled for the client", gaugeMetric, false, extended},
			{"max_easize", "maximum_ea_size_bytes", "Maximum Extended Attribute (EA) size in bytes", gaugeMetric, false, extended},
			{"max_read_ahead_mb", "maximum_read_ahead_megabytes", "Maximum number of megabytes to read ahead", gaugeMetric, false, extended},
			{"max_read_ahead_per_file_mb", "maximum_read_ahead_per_file_megabytes", "Maximum number of megabytes per file to read ahead", gaugeMetric, false, extended},
			{"max_read_ahead_whole_mb", "maximum_read_ahead_whole_megabytes", "Maximum file size in megabytes for a file to be read in its entirety", gaugeMetric, false, extended},
			{"statahead_agl", "statahead_agl_enabled", "Returns '1' if the Asynchronous Glimpse Lock (AGL) for statahead is enabled", gaugeMetric, false, extended},
			{"statahead_max", "statahead_maximum", "Maximum window size for statahead", gaugeMetric, false, extended},
			{"stats", "read_samples_total", readSamplesHelp, counterMetric, false, core},
			{"stats", "read_minimum_size_bytes", readMinimumHelp, gaugeMetric, false, extended},
			{"stats", "read_maximum_size_bytes", readMaximumHelp, gaugeMetric, false, extended},
			{"stats", "read_bytes_total", readTotalHelp, counterMetric, false, core},
			{"stats", "write_samples_total", writeSamplesHelp, counterMetric, false, core},
			{"stats", "write_minimum_size_bytes", writeMinimumHelp, gaugeMetric, false, extended},
			{"stats", "write_maximum_size_bytes", writeMaximumHelp, gaugeMetric, false, extended},
			{"stats", "write_bytes_total", writeTotalHelp, counterMetric, false, core},
			{"stats", "stats_total", statsHelp, counterMetric, true, core},
			{"xattr_cache", "xattr_cache_enabled", "Returns '1' if extended attribute cache is enabled", gaugeMetric, false, extended},
		},
		"mdc/*": {
			{"rpc_stats", "rpcs_in_flight", rpcsInFlightHelp, gaugeMetric, true, core},
		},
		"osc/*": {
			{"rpc_stats", "pages_per_rpc_total", pagesPerRPCHelp, counterMetric, false, core},
			{"rpc_stats", "rpcs_in_flight", rpcsInFlightHelp, gaugeMetric, true, core},
			{"rpc_stats", "rpcs_offset", offsetHelp, gaugeMetric, false, core},
		},
	}
	for path := range metricMap {
		for _, item := range metricMap[path] {
			if filter == extended || item.priorityLevel == core {
				newMetric := newLustreProcMetric(&item, "client", path)
				s.lustreProcMetrics = append(s.lustreProcMetrics, *newMetric)
			}
		}
	}
}

func (s *lustreProcFsSource) generateGenericMetricTemplates(filter string) {
	metricMap := map[string][]lustreHelpStruct{
		"sptlrpc": {
			{"encrypt_page_pools", "physical_pages", physicalPagesHelp, gaugeMetric, false, extended},
			{"encrypt_page_pools", "pages_per_pool", pagesPerPoolHelp, gaugeMetric, false, extended},
			{"encrypt_page_pools", "maximum_pages", maxPagesHelp, gaugeMetric, false, extended},
			{"encrypt_page_pools", "maximum_pools", maxPoolsHelp, gaugeMetric, false, extended},
			{"encrypt_page_pools", "pages_in_pools", totalPagesHelp, gaugeMetric, false, extended},
			{"encrypt_page_pools", "free_pages", totalFreeHelp, gaugeMetric, false, extended},
			{"encrypt_page_pools", "maximum_pages_reached_total", maxPagesReachedHelp, counterMetric, false, extended},
			{"encrypt_page_pools", "grows_total", growsHelp, counterMetric, false, extended},
			{"encrypt_page_pools", "grows_failure_total", growsFailureHelp, counterMetric, false, extended},
			{"encrypt_page_pools", "shrinks_total", shrinksHelp, counterMetric, false, extended},
			{"encrypt_page_pools", "cache_access_total", cacheAccessHelp, counterMetric, false, extended},
			{"encrypt_page_pools", "cache_miss_total", cacheMissingHelp, counterMetric, false, extended},
			{"encrypt_page_pools", "free_page_low", lowFreeMarkHelp, gaugeMetric, false, extended},
			{"encrypt_page_pools", "maximum_waitqueue_depth", maxWaitQueueDepthHelp, gaugeMetric, false, extended},
			{"encrypt_page_pools", "out_of_memory_request_total", outOfMemHelp, counterMetric, false, extended},
		},
	}
	for path := range metricMap {
		for _, item := range metricMap[path] {
			if filter == extended || item.priorityLevel == core {
				newMetric := newLustreProcMetric(&item, "generic", path)
				s.lustreProcMetrics = append(s.lustreProcMetrics, *newMetric)
			}
		}
	}
}

func newLustreProcFsSource() LustreSource {
	var l lustreProcFsSource
	l.basePath = filepath.Join(ProcLocation, "fs/lustre")
	//control which node metrics you pull via flags
	if OstEnabled != disabled {
		l.generateOSTMetricTemplates(OstEnabled)
	}
	if MdtEnabled != disabled {
		l.generateMDTMetricTemplates(MdtEnabled)
	}
	if MgsEnabled != disabled {
		l.generateMGSMetricTemplates(MgsEnabled)
	}
	if MdsEnabled != disabled {
		l.generateMDSMetricTemplates(MdsEnabled)
	}
	if ClientEnabled != disabled {
		l.generateClientMetricTemplates(ClientEnabled)
	}
	if GenericEnabled != disabled {
		l.generateGenericMetricTemplates(GenericEnabled)
	}
	return &l
}

func (s *lustreProcFsSource) Update(ch chan<- prometheus.Metric) (err error) {
	var metricType string
	var directoryDepth int

	for _, metric := range s.lustreProcMetrics {
		directoryDepth = strings.Count(metric.filename, "/")
		paths, err := filepath.Glob(filepath.Join(s.basePath, metric.path, metric.filename))
		if err != nil {
			return err
		}
		if paths == nil {
			continue
		}
		for _, path := range paths {
			metricType = single
			switch metric.filename {
			case "brw_stats", "rpc_stats":
				err = s.parseBRWStats(metric.source, "stats", path, directoryDepth, metric.helpText, metric.promName, metric.hasMultipleVals, func(nodeType string, brwOperation string, brwSize string, nodeName string, name string, helpText string, value float64, extraLabel string, extraLabelValue string) {
					if extraLabelValue == "" {
						ch <- metric.metricFunc([]string{"component", "target", "operation", "size"}, []string{nodeType, nodeName, brwOperation, brwSize}, name, helpText, value)
					} else {
						ch <- metric.metricFunc([]string{"component", "target", "operation", "size", extraLabel}, []string{nodeType, nodeName, brwOperation, brwSize, extraLabelValue}, name, helpText, value)
					}
				})
				if err != nil {
					return err
				}
			case "job_stats":
				err = s.parseJobStats(metric.source, "job_stats", path, directoryDepth, metric.helpText, metric.promName, metric.hasMultipleVals, func(nodeType string, jobid string, nodeName string, name string, helpText string, value float64, extraLabel string, extraLabelValue string) {
					if extraLabelValue == "" {
						ch <- metric.metricFunc([]string{"component", "target", "jobid"}, []string{nodeType, nodeName, jobid}, name, helpText, value)
					} else {
						ch <- metric.metricFunc([]string{"component", "target", "jobid", extraLabel}, []string{nodeType, nodeName, jobid, extraLabelValue}, name, helpText, value)
					}
				})
				if err != nil {
					return err
				}
			default:
				if metric.filename == stats {
					metricType = stats
				} else if metric.filename == mdStats {
					metricType = mdStats
				} else if metric.filename == encryptPagePools {
					metricType = encryptPagePools
				}
				err = s.parseFile(metric.source, metricType, path, directoryDepth, metric.helpText, metric.promName, metric.hasMultipleVals, func(nodeType string, nodeName string, name string, helpText string, value float64, extraLabel string, extraLabelValue string) {
					if extraLabelValue == "" {
						ch <- metric.metricFunc([]string{"component", "target"}, []string{nodeType, nodeName}, name, helpText, value)
					} else {
						ch <- metric.metricFunc([]string{"component", "target", extraLabel}, []string{nodeType, nodeName, extraLabelValue}, name, helpText, value)
					}
				})
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func getStatsOperationMetrics(statsFile string, promName string, helpText string) (metricList []lustreStatsMetric, err error) {
	operationSlice := []multistatParsingStruct{
		{pattern: "open", index: 1},
		{pattern: "close", index: 1},
		{pattern: "getattr", index: 1},
		{pattern: "setattr", index: 1},
		{pattern: "getxattr", index: 1},
		{pattern: "setxattr", index: 1},
		{pattern: "statfs", index: 1},
		{pattern: "seek", index: 1},
		{pattern: "readdir", index: 1},
		{pattern: "truncate", index: 1},
		{pattern: "alloc_inode", index: 1},
		{pattern: "removexattr", index: 1},
		{pattern: "unlink", index: 1},
		{pattern: "inode_permission", index: 1},
		{pattern: "create", index: 1},
		{pattern: "get_info", index: 1},
		{pattern: "set_info_async", index: 1},
		{pattern: "connect", index: 1},
		{pattern: "ping", index: 1},
	}
	for _, operation := range operationSlice {
		opStat := regexCaptureString(operation.pattern+" .*", statsFile)
		if len(opStat) < 1 {
			continue
		}
		r, err := regexp.Compile(" +")
		if err != nil {
			continue
		}
		bytesSplit := r.Split(opStat, -1)
		result, err := strconv.ParseFloat(bytesSplit[operation.index], 64)
		if err != nil {
			return nil, err
		}
		metricList = append(metricList, *newLustreStatsMetric(promName, helpText, result, "operation", operation.pattern))
	}
	return metricList, nil
}

func getStatsIOMetrics(statsFile string, promName string, helpText string) (metricList []lustreStatsMetric, err error) {
	// bytesSplit is in the following format:
	// bytesString: {name} {number of samples} 'samples' [{units}] {minimum} {maximum} {sum}
	// bytesSplit:   [0]    [1]                 [2]       [3]       [4]       [5]       [6]
	bytesMap := map[string]multistatParsingStruct{
		readSamplesHelp:       {pattern: "read_bytes .*", index: 1},
		readMinimumHelp:       {pattern: "read_bytes .*", index: 4},
		readMaximumHelp:       {pattern: "read_bytes .*", index: 5},
		readTotalHelp:         {pattern: "read_bytes .*", index: 6},
		writeSamplesHelp:      {pattern: "write_bytes .*", index: 1},
		writeMinimumHelp:      {pattern: "write_bytes .*", index: 4},
		writeMaximumHelp:      {pattern: "write_bytes .*", index: 5},
		writeTotalHelp:        {pattern: "write_bytes .*", index: 6},
		physicalPagesHelp:     {pattern: "physical pages: .*", index: 2},
		pagesPerPoolHelp:      {pattern: "pages per pool: .*", index: 3},
		maxPagesHelp:          {pattern: "max pages: .*", index: 2},
		maxPoolsHelp:          {pattern: "max pools: .*", index: 2},
		totalPagesHelp:        {pattern: "total pages: .*", index: 2},
		totalFreeHelp:         {pattern: "total free: .*", index: 2},
		maxPagesReachedHelp:   {pattern: "max pages reached: .*", index: 3},
		growsHelp:             {pattern: "grows: .*", index: 1},
		growsFailureHelp:      {pattern: "grows failure: .*", index: 2},
		shrinksHelp:           {pattern: "shrinks: .*", index: 1},
		cacheAccessHelp:       {pattern: "cache access: .*", index: 2},
		cacheMissingHelp:      {pattern: "cache missing: .*", index: 2},
		lowFreeMarkHelp:       {pattern: "low free mark: .*", index: 3},
		maxWaitQueueDepthHelp: {pattern: "max waitqueue depth: .*", index: 3},
		outOfMemHelp:          {pattern: "out of mem: .*", index: 3},
	}
	pattern := bytesMap[helpText].pattern
	bytesString := regexCaptureString(pattern, statsFile)
	if len(bytesString) < 1 {
		return nil, nil
	}
	r, err := regexp.Compile(" +")
	if err != nil {
		return nil, err
	}
	bytesSplit := r.Split(bytesString, -1)
	result, err := strconv.ParseFloat(bytesSplit[bytesMap[helpText].index], 64)
	if err != nil {
		return nil, err
	}
	metricList = append(metricList, *newLustreStatsMetric(promName, helpText, result, "", ""))

	return metricList, nil
}

func splitBRWStats(statBlock string) (metricList []lustreBRWMetric, err error) {
	if len(statBlock) == 0 || statBlock == "" {
		return nil, nil
	}

	// Skip the first line of text as it doesn't contain any metrics
	for _, line := range strings.Split(statBlock, "\n")[1:] {
		if len(line) > 1 {
			fields := strings.Fields(line)
			// Lines are in the following format:
			// [size] [# read RPCs] [relative read size (%)] [cumulative read size (%)] | [# write RPCs] [relative write size (%)] [cumulative write size (%)]
			// [0]    [1]           [2]                      [3]                       [4] [5]           [6]                       [7]
			if len(fields) >= 6 {
				size, readRPCs, writeRPCs := fields[0], fields[1], fields[5]
				size = strings.Replace(size, ":", "", -1)
				metricList = append(metricList, lustreBRWMetric{size: size, operation: "read", value: readRPCs})
				metricList = append(metricList, lustreBRWMetric{size: size, operation: "write", value: writeRPCs})
			} else if len(fields) >= 1 {
				size, rpcs := fields[0], fields[1]
				size = strings.Replace(size, ":", "", -1)
				metricList = append(metricList, lustreBRWMetric{size: size, operation: "read", value: rpcs})
			} else {
				continue
			}
		}
	}
	return metricList, nil
}

func parseStatsFile(helpText string, promName string, path string, hasMultipleVals bool) (metricList []lustreStatsMetric, err error) {
	statsFileBytes, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	statsFile := string(statsFileBytes[:])
	var statsList []lustreStatsMetric
	if hasMultipleVals {
		statsList, err = getStatsOperationMetrics(statsFile, promName, helpText)
	} else {
		statsList, err = getStatsIOMetrics(statsFile, promName, helpText)
	}
	if err != nil {
		return nil, err
	}
	if statsList != nil {
		metricList = append(metricList, statsList...)
	}

	return metricList, nil
}

func getJobStatsIOMetrics(jobBlock string, jobID string, promName string, helpText string) (metricList []lustreJobsMetric, err error) {
	// opMap matches the given helpText value with the placement of the numeric fields within each metric line.
	// For example, the number of samples is the first number in the line and has a helpText of readSamplesHelp,
	// hence the 'index' value of 0. 'pattern' is the regex capture pattern for the desired line.
	opMap := map[string]multistatParsingStruct{
		readSamplesHelp:  {index: 0, pattern: "read_bytes"},
		readMinimumHelp:  {index: 1, pattern: "read_bytes"},
		readMaximumHelp:  {index: 2, pattern: "read_bytes"},
		readTotalHelp:    {index: 3, pattern: "read_bytes"},
		writeSamplesHelp: {index: 0, pattern: "write_bytes"},
		writeMinimumHelp: {index: 1, pattern: "write_bytes"},
		writeMaximumHelp: {index: 2, pattern: "write_bytes"},
		writeTotalHelp:   {index: 3, pattern: "write_bytes"},
	}
	// If the metric isn't located in the map, don't try to parse a value for it.
	if _, exists := opMap[helpText]; !exists {
		return nil, nil
	}
	pattern := opMap[helpText].pattern
	opStat := regexCaptureString(pattern+": .*", jobBlock)
	opNumbers := regexCaptureNumbers(opStat)
	if len(opNumbers) < 1 {
		return nil, nil
	}
	result, err := strconv.ParseFloat(strings.TrimSpace(opNumbers[opMap[helpText].index]), 64)
	if err != nil {
		return nil, err
	}
	if result == 0 {
		return nil, nil
	}
	metricList = append(metricList,
		lustreJobsMetric{jobID: jobID,
			lustreStatsMetric: *newLustreStatsMetric(promName, helpText, result, "", ""),
		})

	return metricList, err
}

func getJobNum(jobBlock string) (jobID string, err error) {
	jobID = regexCaptureJobid(jobBlock)
	if jobID == "" {
		return "", errors.New("No valid jobid found in block: " + jobBlock)
	}
	return strings.TrimSpace(jobID), nil
}

func getJobStatsOperationMetrics(jobBlock string, jobID string, promName string, helpText string) (metricList []lustreJobsMetric, err error) {
	operationSlice := []multistatParsingStruct{
		{index: 0, pattern: "open"},
		{index: 0, pattern: "close"},
		{index: 0, pattern: "mknod"},
		{index: 0, pattern: "link"},
		{index: 0, pattern: "unlink"},
		{index: 0, pattern: "mkdir"},
		{index: 0, pattern: "rmdir"},
		{index: 0, pattern: "rename"},
		{index: 0, pattern: "getattr"},
		{index: 0, pattern: "setattr"},
		{index: 0, pattern: "getxattr"},
		{index: 0, pattern: "setxattr"},
		{index: 0, pattern: "statfs"},
		{index: 0, pattern: "sync"},
		{index: 0, pattern: "samedir_rename"},
		{index: 0, pattern: "crossdir_rename"},
		{index: 0, pattern: "punch"},
		{index: 0, pattern: "destroy"},
		{index: 0, pattern: "create"},
		{index: 0, pattern: "get_info"},
		{index: 0, pattern: "set_info"},
		{index: 0, pattern: "quotactl"},
	}
	for _, operation := range operationSlice {
		opStat := regexCaptureString(operation.pattern+": .*", jobBlock)
		opNumbers := regexCaptureStrings("[0-9]*\\.[0-9]+|[0-9]+", opStat)
		if len(opNumbers) < 1 {
			continue
		}
		var result float64
		result, err = strconv.ParseFloat(strings.TrimSpace(opNumbers[operation.index]), 64)
		if err != nil {
			return nil, err
		}
		if result == 0 {
			continue
		}
		metricList = append(metricList,
			lustreJobsMetric{jobID: jobID,
				lustreStatsMetric: *newLustreStatsMetric(promName, helpText, result, "operation", operation.pattern),
			})
	}
	return metricList, err
}

func parseJobStatsText(jobStats string, promName string, helpText string, hasMultipleVals bool) (metricList []lustreJobsMetric, err error) {
	jobs := regexCaptureJobStats(jobStats)
	if len(jobs) < 1 {
		return nil, nil
	}
	var jobList []lustreJobsMetric
	for _, job := range jobs {
		jobID, err := getJobNum(job)
		if err != nil {
			log.Error(err)
			continue
		}
		if hasMultipleVals {
			jobList, err = getJobStatsOperationMetrics(job, jobID, promName, helpText)
		} else {
			jobList, err = getJobStatsIOMetrics(job, jobID, promName, helpText)
		}
		if err != nil {
			return nil, err
		}
		if jobList != nil {
			metricList = append(metricList, jobList...)
		}
	}
	return metricList, nil
}

func (s *lustreProcFsSource) parseJobStats(nodeType string, metricType string, path string, directoryDepth int, helpText string, promName string, hasMultipleVals bool, handler func(string, string, string, string, string, float64, string, string)) (err error) {
	_, nodeName, err := parseFileElements(path, directoryDepth)
	if err != nil {
		return err
	}
	jobStatsBytes, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return err
	}

	jobStatsFile := string(jobStatsBytes[:])

	metricList, err := parseJobStatsText(jobStatsFile, promName, helpText, hasMultipleVals)
	if err != nil {
		return err
	}

	for _, item := range metricList {
		handler(nodeType, item.jobID, nodeName, item.lustreStatsMetric.title, item.lustreStatsMetric.help, item.lustreStatsMetric.value, item.lustreStatsMetric.extraLabel, item.lustreStatsMetric.extraLabelValue)
	}
	return nil
}

func (s *lustreProcFsSource) parseBRWStats(nodeType string, metricType string, path string, directoryDepth int, helpText string, promName string, hasMultipleVals bool, handler func(string, string, string, string, string, string, float64, string, string)) (err error) {
	_, nodeName, err := parseFileElements(path, directoryDepth)
	if err != nil {
		return err
	}
	metricBlocks := map[string]string{
		pagesPerBlockRWHelp:    "pages per bulk r/w",
		discontiguousPagesHelp: "discontiguous pages",
		diskIOsInFlightHelp:    "disk I/Os in flight",
		ioTimeHelp:             "I/O time",
		diskIOSizeHelp:         "disk I/O size",
		pagesPerRPCHelp:        "pages per rpc",
		rpcsInFlightHelp:       "rpcs in flight",
		offsetHelp:             "offset",
	}
	statsFileBytes, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return err
	}
	statsFile := string(statsFileBytes[:])
	block := regexCaptureString("(?ms:^"+metricBlocks[helpText]+".*?(\n\n|\\z))", statsFile)
	metricList, err := splitBRWStats(block)
	if err != nil {
		return err
	}
	extraLabel := ""
	extraLabelValue := ""
	if hasMultipleVals {
		extraLabel = "type"
		pathElements := strings.Split(path, "/")
		extraLabelValue = pathElements[len(pathElements)-3]
	}
	for _, item := range metricList {
		value, err := strconv.ParseFloat(item.value, 64)
		if err != nil {
			return err
		}
		handler(nodeType, item.operation, convertToBytes(item.size), nodeName, promName, helpText, value, extraLabel, extraLabelValue)
	}
	return nil
}

func (s *lustreProcFsSource) parseFile(nodeType string, metricType string, path string, directoryDepth int, helpText string, promName string, hasMultipleVals bool, handler func(string, string, string, string, float64, string, string)) (err error) {
	_, nodeName, err := parseFileElements(path, directoryDepth)
	if err != nil {
		return err
	}
	switch metricType {
	case single:
		value, err := os.ReadFile(filepath.Clean(path))
		if err != nil {
			return err
		}
		convertedValue, err := strconv.ParseFloat(strings.TrimSpace(string(value)), 64)
		if err != nil {
			return err
		}
		handler(nodeType, nodeName, promName, helpText, convertedValue, "", "")
	case stats, mdStats, encryptPagePools:
		metricList, err := parseStatsFile(helpText, promName, path, hasMultipleVals)
		if err != nil {
			return err
		}

		for _, metric := range metricList {
			handler(nodeType, nodeName, metric.title, metric.help, metric.value, metric.extraLabel, metric.extraLabelValue)
		}
	}
	return nil
}
