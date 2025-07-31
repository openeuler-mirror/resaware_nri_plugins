package prometheus

import (
	"io/ioutil"
	"k8s.io/klog"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const defaultMetricCollectPeriodSecond = 10

// 定义 Prometheus 指标
var (
	mpamL3Occupancy = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "mpam_l3_occupancy",
			Help: "L3 cache occupancy in bytes",
		},
		[]string{"group", "cache_id"},
	)

	mpamMBMTotalBytes = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "mpam_mbm_total_bytes",
			Help: "Memory bandwidth usage in bytes",
		},
		[]string{"group", "numa_id"},
	)

	excludeDirs = map[string]bool{
		"info":       true,
		"mon_data":   true,
		"mon_groups": true,
	}
	rootResctrlPath = os.Getenv("RESCTRL_PATH")

	metricPort                   = os.Getenv("METRIC_PORT")
	metricCollectPeriodSecond, _ = strconv.Atoi(os.Getenv("METRIC_COLLECT_PERIOD"))
)

func init() {
	if rootResctrlPath == "" {
		rootResctrlPath = "/sys/fs/resctrl"
	}
	if metricPort == "" {
		metricPort = "8000"
	}
	if metricCollectPeriodSecond <= 0 || metricCollectPeriodSecond > 60 {
		metricCollectPeriodSecond = defaultMetricCollectPeriodSecond
	}
}

func collectMetrics() {
	groups, err := ioutil.ReadDir(rootResctrlPath)
	if err != nil {
		klog.Errorf("Error reading mon_data directory: %v", err)
		return
	}

	// 处理根组
	processGroup(rootResctrlPath, "total")

	// 过滤掉不需要的目录

	for _, group := range groups {
		if excludeDirs[group.Name()] {
			continue
		}
		groupPath := filepath.Join(rootResctrlPath, group.Name())
		if !group.IsDir() {
			continue
		}
		// 处理其他监控组
		processGroup(groupPath, group.Name())
	}
}

func processGroup(groupPath, groupName string) {
	groupMonDataPath := filepath.Join(groupPath, "mon_data")
	resources, err := ioutil.ReadDir(path.Join(groupMonDataPath))
	if err != nil {
		klog.Errorf("Error reading group directory %s: %v", groupMonDataPath, err)
		return
	}

	for _, resource := range resources {
		if excludeDirs[resource.Name()] {
			continue
		}

		resourcePath := filepath.Join(groupMonDataPath, resource.Name())
		if !resource.IsDir() {
			continue
		}

		files, err := ioutil.ReadDir(resourcePath)
		if err != nil {
			klog.Errorf("Error reading resource directory %s: %v", resourcePath, err)
			continue
		}

		for _, file := range files {
			filePath := filepath.Join(resourcePath, file.Name())
			if file.Name() == "llc_occupancy" {
				data, err := ioutil.ReadFile(filePath)
				if err != nil {
					klog.Errorf("Error reading file %s: %v", filePath, err)
					continue
				}
				value, _ := strconv.ParseFloat(strings.TrimSpace(string(data)), 64)
				cacheID := strings.Split(resource.Name(), "_")[2]
				mpamL3Occupancy.WithLabelValues(groupName, cacheID).Set(value)
			} else if file.Name() == "mbm_total_bytes" {
				data, err := ioutil.ReadFile(filePath)
				if err != nil {
					klog.Errorf("Error reading file %s: %v", filePath, err)
					continue
				}
				value, _ := strconv.ParseFloat(strings.TrimSpace(string(data)), 64)
				numaID := strings.Split(resource.Name(), "_")[2]
				mpamMBMTotalBytes.WithLabelValues(groupName, numaID).Set(value)
			}
		}
	}
}

func StartMetricsServer() {
	// 启动 HTTP 服务器，暴露 Prometheus 指标
	customRegistry := prometheus.NewRegistry()

	customRegistry.MustRegister(mpamL3Occupancy)
	customRegistry.MustRegister(mpamMBMTotalBytes)

	http.Handle("/metrics", promhttp.HandlerFor(customRegistry, promhttp.HandlerOpts{}))
	go func() {
		klog.Fatal(http.ListenAndServe(":8000", nil))
	}()

	// 定时采集指标
	ticker := time.NewTicker(10 * time.Second)
	for {
		select {
		case <-ticker.C:
			collectMetrics()
		}
	}
}
