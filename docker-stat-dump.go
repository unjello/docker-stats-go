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
)

var dockerAPIEndpoint = `127.0.0.1:2375`

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

// Stats .
type Stats struct {
	container DockerContainer
	stats     DockerStats
}

func main() {
	// TODO: Add docker-endpoint param
	// TODO: Add sleep interval param
	// TODO: Add formatting param
	quit := make(chan error)
	done := make(chan int)
	stat := make(chan Stats)

	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		quit <- fmt.Errorf("SIGTERM Received")
	}()

	dockerContainerList, err := getDockerContainerList()
	if err != nil {
		// TODO: Fix it to break out of a loop maybe?
		os.Exit(0)
	}

	dockerMonitors := len(dockerContainerList)
	for i := 0; i < len(dockerContainerList); i++ {
		// TODO: Make this a Encoding/Writer
		go func(index int) {
			dockerStats, err := getDockerContainerStats(dockerContainerList[index])
			if err != nil {
				// TODO: handle error better
				return
			}
			stat <- Stats{container: dockerContainerList[index], stats: *dockerStats}
			done <- index
		}(i)
	}

	for {
		select {
		case s := <-stat:
			fmt.Printf("%s, %s, %d\n", s.container.ID[:10], s.container.Names[0], s.stats.CPUStats.CPUUsage.TotalUsage)

		case <-done:
			dockerMonitors--
			if dockerMonitors == 0 {
				os.Exit(0)
			}
		case <-quit:
			// TODO: handle exit with some message?
			os.Exit(0)
		}
	}
	//time.Sleep(time.Duration(10) * time.Millisecond)
	//}
}
