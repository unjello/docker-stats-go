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
