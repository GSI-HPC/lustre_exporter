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

package main

import (
	_ "embed"
	"fmt"
	stdlog "log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/GSI-HPC/lustre_exporter/sources"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	scrapeDurations = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace: sources.Namespace,
			Subsystem: "exporter",
			Name:      "scrape_duration_seconds",
			Help:      "lustre_exporter: Duration of a scrape job.",
		},
		[]string{"source", "result"},
	)
	//go:embed VERSION
	exporterVersion string
)

// LustreSource is a list of all sources that the user would like to collect.
type LustreSource struct {
	sourceList map[string]sources.LustreSource
}

// Describe implements the prometheus.Describe interface
func (l LustreSource) Describe(ch chan<- *prometheus.Desc) {
	scrapeDurations.Describe(ch)
}

// Collect implements the prometheus.Collect interface
func (l LustreSource) Collect(ch chan<- prometheus.Metric) {
	wg := sync.WaitGroup{}
	wg.Add(len(l.sourceList))
	for name, c := range l.sourceList {
		go func(name string, c sources.LustreSource) {
			collectFromSource(name, c, ch)
			wg.Done()
		}(name, c)
	}
	wg.Wait()
	scrapeDurations.Collect(ch)
}

func collectFromSource(name string, s sources.LustreSource, ch chan<- prometheus.Metric) {
	result := "success"
	begin := time.Now()
	err := s.Update(ch)
	duration := time.Since(begin)
	if err != nil {
		log.Errorf("source %q failed after %f seconds - %s", name, duration.Seconds(), err)
		result = "error"
	} else {
		log.Debugf("source %q succeeded after %f seconds", name, duration.Seconds())
	}
	scrapeDurations.WithLabelValues(name, result).Observe(duration.Seconds())
}

func loadSources(list []string) (map[string]sources.LustreSource, []error) {
	sourceList := map[string]sources.LustreSource{}
	var errList []error
	for _, name := range list {
		fn, ok := sources.Factories[name]
		if ok {
			if c := fn(); c != nil {
				sourceList[name] = c
				continue
			}
		}
		errList = append(errList, fmt.Errorf("source %q not available", name))
	}
	return sourceList, errList
}

func initLogFile(path string) {
	logFile, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		log.Fatal(err)
	}
	log.SetOutput(logFile)
}

func main() {

	kingpin.HelpFlag.Short('h')

	var (
		clientEnabled       = kingpin.Flag("collector.client", "Set client metric level. Valid levels: [extended, core, disabled]").Default("extended").Enum("extended", "core", "disabled")
		genericEnabled      = kingpin.Flag("collector.generic", "Set generic metric level. Valid levels: [extended, core, disabled]").Default("extended").Enum("extended", "core", "disabled")
		lnetEnabled         = kingpin.Flag("collector.lnet", "Set LNET metric level. Valid levels: [extended, core, disabled]").Default("extended").Enum("extended", "core", "disabled")
		mdsEnabled          = kingpin.Flag("collector.mds", "Set MDS metric level. Valid levels: [extended, core, disabled]").Default("extended").Enum("extended", "core", "disabled")
		mdtEnabled          = kingpin.Flag("collector.mdt", "Set MDT metric level. Valid levels: [extended, core, disabled]").Default("extended").Enum("extended", "core", "disabled")
		mgsEnabled          = kingpin.Flag("collector.mgs", "Set MGS metric level. Valid levels: [extended, core, disabled]").Default("extended").Enum("extended", "core", "disabled")
		ostEnabled          = kingpin.Flag("collector.ost", "Set OST metric level. Valid levels: [extended, core, disabled]").Default("extended").Enum("extended", "core", "disabled")
		healthStatusEnabled = kingpin.Flag("collector.health", "Set Health metric level. Valid levels: [extended, core, disabled]").Default("extended").Enum("extended", "core", "disabled")
		listenAddress       = kingpin.Flag("web.listen-address", "Address to use to expose Lustre metrics.").Default(":9169").String()
		metricsPath         = kingpin.Flag("web.telemetry-path", "Path to use to expose Lustre metrics.").Default("/metrics").String()
		logLevel            = kingpin.Flag("log.level", "Set log level. Valid levels: [debug, info, warn, error]").Default("info").Enum("debug", "info", "warn", "error")
		logFile             = kingpin.Flag("log.file", "Redirect log output to specified file.").Default("").String()
		printVersion        = kingpin.Flag("version", "Print version.").Short('v').Bool()
	)

	kingpin.Parse()

	if *printVersion {
		fmt.Print(exporterVersion)
		os.Exit(0)
	}

	var level, _ = log.ParseLevel(*logLevel)
	log.SetLevel(level)
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})

	if *logFile != "" {
		log.Info("Redirecting log output to file: ", *logFile)
		initLogFile(*logFile)
	}

	log.Info("Starting...")

	log.Infof("Collector status:")
	sources.OstEnabled = *ostEnabled
	log.Infof(" - OST State: %s", sources.OstEnabled)
	sources.MdtEnabled = *mdtEnabled
	log.Infof(" - MDT State: %s", sources.MdtEnabled)
	sources.MgsEnabled = *mgsEnabled
	log.Infof(" - MGS State: %s", sources.MgsEnabled)
	sources.MdsEnabled = *mdsEnabled
	log.Infof(" - MDS State: %s", sources.MdsEnabled)
	sources.ClientEnabled = *clientEnabled
	log.Infof(" - Client State: %s", sources.ClientEnabled)
	sources.GenericEnabled = *genericEnabled
	log.Infof(" - Generic State: %s", sources.GenericEnabled)
	sources.LnetEnabled = *lnetEnabled
	log.Infof(" - LNET State: %s", sources.LnetEnabled)
	sources.HealthStatusEnabled = *healthStatusEnabled
	log.Infof(" - Health State: %s", sources.HealthStatusEnabled)

	enabledSources := []string{"procfs", "sys", "sysfs", "lctl"}

	sourceList, errList := loadSources(enabledSources)

	if errList != nil {
		for _, err := range errList {
			log.Errorf("Couldn't load source: %s", err)
		}
		log.Fatal("Unable to load sources")
	}

	log.Infof("Available sources:")

	for s := range sourceList {
		log.Infof(" - %s", s)
	}

	prometheus.MustRegister(LustreSource{sourceList: sourceList})
	//load InstrumentMetricHandler
	handler := promhttp.InstrumentMetricHandler(prometheus.DefaultRegisterer,
		promhttp.HandlerFor(prometheus.DefaultGatherer,
			promhttp.HandlerOpts{
				ErrorLog: stdlog.New(os.Stderr, "", stdlog.LstdFlags)}))

	http.Handle(*metricsPath, handler)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var num int
		num, err := w.Write([]byte(`<html>
			<head><title>Lustre Exporter</title></head>
			<body>
			<h1>Lustre Exporter</h1>
			<p><a href="` + *metricsPath + `">Metrics</a></p>
			</body>
			</html>`))
		if err != nil {
			log.Fatal(num, err)
		}
	})

	log.Info("Listening on", *listenAddress)
	err := http.ListenAndServe(*listenAddress, nil)
	if err != nil {
		log.Fatal("Error on Listen", err)
	}
}
