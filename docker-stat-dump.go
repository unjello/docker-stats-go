/// docker-stats-dump by andrzej lichnerowicz, unlicensed (~public domain)
/// program uses API v1.33 https://docs.docker.com/engine/api/v1.33/
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

// TODO: make it a method
func getDockerContainerStats(context context.Context, client *client.Client, container types.Container) (*types.Stats, error) {
	// FIXME: do proper streaming

	// TODO: test for missing container
	response, err := client.ContainerStats(context, container.ID, false)
	if err != nil {
		return nil, err
	}

	responseData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	var dockerStats types.Stats
	json.Unmarshal(responseData, &dockerStats)
	return &dockerStats, nil
}

// Stats .
type Stats struct {
	container types.Container
	stats     types.Stats
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

	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	dockerContainerList, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		// TODO: Fix it to break out of a loop maybe?
		os.Exit(0)
	}

	dockerMonitors := len(dockerContainerList)
	for i := 0; i < len(dockerContainerList); i++ {
		// TODO: Make this a Encoding/Writer
		go func(index int) {
			dockerStats, err := getDockerContainerStats(context.Background(), cli, dockerContainerList[index])
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
				go func() {
					quit <- fmt.Errorf("No monitors left")
				}()
			}
		case <-quit:
			// TODO: handle exit with some message?
			os.Exit(0)
		}
	}
	//time.Sleep(time.Duration(10) * time.Millisecond)
	//}
}
