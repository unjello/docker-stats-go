/// docker-stats-dump by andrzej lichnerowicz, unlicensed (~public domain)
/// program uses API v1.33 https://docs.docker.com/engine/api/v1.33/
package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var dockerAPIEndpoint = `127.0.0.1:2375`

// DockerContainer .
type DockerContainer struct {
	ID      string
	Names   []string
	Image   string
	ImageID string
}

type DockerStats struct {
	Read      time.Time
	PidsStats struct {
		Current int64
	} `json:"pids_stats"`
	Networks map[string]struct {
		RxBytes   int64 `json:"rx_bytes"`
		RxDropped int64 `json:"rx_dropped"`
		RxErrors  int64 `json:"rx_errors"`
		RxPackets int64 `json:"rx_packets"`
		TxBytes   int64 `json:"tx_bytes"`
		TxDropped int64 `json:"tx_dropped"`
		TxErrors  int64 `json:"tx_errors"`
		TxPackets int64 `json:"tx_packets"`
	}
	MemoryStats struct {
		Stats struct {
			TotalPgmajfault         int64  `json:"total_pgmajfault"`
			Cache                   int64  `json:"cache"`
			MappedFile              int64  `json:"mapped_file"`
			TotalInactiveFile       int64  `json:"total_inactive_file"`
			Pgpgout                 int64  `json:"pgpgout"`
			Rss                     int64  `json:"rss"`
			TotalMappedFile         int64  `json:"total_mapped_file"`
			Writeback               int64  `json:"writeback"`
			Unevictable             int64  `json:"unevictable"`
			Pgpgin                  int64  `json:"pgpgin"`
			TotalUnevictable        int64  `json:"total_unevictable"`
			Pgmajfault              int64  `json:"pgmajfault"`
			TotalRss                int64  `json:"total_rss"`
			TotalRssHuge            int64  `json:"total_rss_huge"`
			TotalWriteback          int64  `json:"total_writeback"`
			TotalInactiveAnon       int64  `json:"total_inactive_anon"`
			RssHuge                 int64  `json:"rss_huge"`
			HierarchicalMemoryLimit uint64 `json:"hierarchical_memory_limit"`
			TotalPgfault            int64  `json:"total_pgfault"`
			TotalActiveFile         int64  `json:"total_active_file"`
			ActiveAnon              int64  `json:"active_anon"`
			TotalActiveAnon         int64  `json:"total_active_anon"`
			TotalPgpgout            int64  `json:"total_pgpgout"`
			TotalCache              int64  `json:"total_cache"`
			InactiveAnon            int64  `json:"inactive_anon"`
			ActiveFile              int64  `json:"active_file"`
			Pgfault                 int64  `json:"pgfault"`
			InactiveFile            int64  `json:"inactive_file"`
			TotalPgpgin             int64  `json:"total_pgpgin"`
		}
		MaxUsage  int64 `json:"max_usage"`
		Usage     int64
		FailCount int64 `json:"failcnt"`
		Limit     int64
	} `json:"memory_stats"`
	BlkioStats struct {
		IoServiceBytesRecursive []struct {
			Major int64  `json:"major"`
			Minor int64  `json:"minor"`
			Op    string `json:"op"`
			Value int64  `json:"value"`
		} `json:"io_service_bytes_recursive"`
		IoServicedRecursive []struct {
			Major int64  `json:"major"`
			Minor int64  `json:"minor"`
			Op    string `json:"op"`
			Value int64  `json:"value"`
		} `json:"io_serviced_recursive"`
		IoQueueRecursive []struct {
			Major int64  `json:"major"`
			Minor int64  `json:"minor"`
			Op    string `json:"op"`
			Value int64  `json:"value"`
		} `json:"io_queue_recursive"`
		IoServiceTimeRecursive []struct {
			Major int64  `json:"major"`
			Minor int64  `json:"minor"`
			Op    string `json:"op"`
			Value int64  `json:"value"`
		} `json:"io_service_time_recursive"`
		IoWaitTimeRecursive []struct {
			Major int64  `json:"major"`
			Minor int64  `json:"minor"`
			Op    string `json:"op"`
			Value int64  `json:"value"`
		} `json:"io_wait_time_recursive"`
		IoMergedRecursive []struct {
			Major int64  `json:"major"`
			Minor int64  `json:"minor"`
			Op    string `json:"op"`
			Value int64  `json:"value"`
		} `json:"io_merged_recursive"`
		IoTimeRecursive []struct {
			Major int64  `json:"major"`
			Minor int64  `json:"minor"`
			Op    string `json:"op"`
			Value int64  `json:"value"`
		} `json:"io_time_recursive"`
		SectorsRecursive []struct {
			Major int64  `json:"major"`
			Minor int64  `json:"minor"`
			Op    string `json:"op"`
			Value int64  `json:"value"`
		} `json:"sectors_recursive"`
	} `json:"blkio_stats"`
	CPUStats struct {
		CPUUsage struct {
			PercpuUsage       []int64 `json:"percpu_usage"`
			UsageInUsermode   int64   `json:"usage_in_usermode"`
			TotalUsage        int64   `json:"total_usage"`
			UsageInKernelmode int64   `json:"usage_in_kernelmode"`
		} `json:"cpu_usage"`
		SystemCPUUsage int64 `json:"system_cpu_usage"`
		ThrottlingData struct {
			Periods          int64 `json:"periods"`
			ThrottledPeriods int64 `json:"throttled_periods"`
			ThrottledTime    int64 `json:"throttled_time"`
		} `json:"throttling_data"`
	} `json:"cpu_stats"`
	PrecpuStats struct {
		CPUUsage struct {
			PercpuUsage       []int64 `json:"percpu_usage"`
			UsageInUsermode   int64   `json:"usage_in_usermode"`
			TotalUsage        int64   `json:"total_usage"`
			UsageInKernelmode int64   `json:"usage_in_kernelmode"`
		} `json:"cpu_usage"`
		SystemCPUUsage int64 `json:"system_cpu_usage"`
		ThrottlingData struct {
			Periods          int64 `json:"periods"`
			ThrottledPeriods int64 `json:"throttled_periods"`
			ThrottledTime    int64 `json:"throttled_time"`
		} `json:"throttling_data"`
	} `json:"precpu_stats"`
}

func getResponseBody(url string) ([]byte, error) {
	response, err := http.Get(url)
	if err != nil {
		log.Fatal(err.Error())
		return nil, err
	}

	responseData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	return responseData, nil
}

// TODO: make it a method
func getDockerContainerList() ([]DockerContainer, error) {
	url := fmt.Sprintf("http://%s/containers/json", dockerAPIEndpoint)

	response, err := getResponseBody(url)
	if err != nil {
		return nil, err
	}

	// top level array trick goes to
	// https://coderwall.com/p/4c2zig/decode-top-level-json-array-into-a-slice-of-structs-in-golang
	dockerContainersList := make([]DockerContainer, 0)
	json.Unmarshal(response, &dockerContainersList)

	return dockerContainersList, nil
}

// TODO: make it a method
func getDockerContainerStats(container DockerContainer) (*DockerStats, error) {
	// FIXME: do proper streaming
	url := fmt.Sprintf("http://%s/containers/%s/stats?stream=false", dockerAPIEndpoint, container.ID)

	// TODO: test for missing container
	response, err := getResponseBody(url)
	if err != nil {
		return nil, err
	}

	var dockerStats DockerStats
	json.Unmarshal(response, &dockerStats)
	return &dockerStats, nil
}

func main() {
	// TODO: Add docker-endpoint param
	// TODO: Add sleep interval param
	// TODO: Add formatting param
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		os.Exit(1)
	}()

	fmt.Println("id, time, inverval")
	//for {

	dockerContainerList, err := getDockerContainerList()
	if err != nil {
		// TODO: Fix it to break out of a loop maybe?
		os.Exit(0)
	}

	for i := 0; i < len(dockerContainerList); i++ {
		// TODO: Make this a Encoding/Writer
		fmt.Println(dockerContainerList[i].ID)
		dockerStats, err := getDockerContainerStats(dockerContainerList[i])
		if err != nil {
			// TODO: handle error better
			os.Exit(0)
		}
		fmt.Printf("%+v", dockerStats)

	}
	//time.Sleep(time.Duration(10) * time.Millisecond)
	//}
}
