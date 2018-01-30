/// docker-stats-dump by andrzej lichnerowicz, unlicensed (~public domain)
/// program uses API v1.33 https://docs.docker.com/engine/api/v1.33/
package main

import (
	"bufio"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

// TODO: make it a method
func getDockerContainerStats(context context.Context, client *client.Client, stat chan<- Stats, container types.Container) error {
	// TODO: test for missing container
	response, err := client.ContainerStats(context, container.ID, true)
	if err != nil {
		return err
	}

	reader := bufio.NewReader(response.Body)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			return fmt.Errorf("Stream finished")
		}
		var dockerStats types.Stats
		json.Unmarshal(line, &dockerStats)
		stat <- Stats{container: container, stats: dockerStats, os: response.OSType}
	}
}

// Stats .
type Stats struct {
	container types.Container
	stats     types.Stats
	os        string
}

type CalculatedStats struct {
	OS               string
	ID               string
	Name             string
	CpuPercentage    float64
	Memory           float64
	MemoryLimit      float64
	MemoryPercentage float64
}

func (cs *CalculatedStats) Strings() []string {
	var t []string

	t = append(t, cs.OS)
	t = append(t, cs.ID[:10])
	t = append(t, strings.TrimLeft(cs.Name, "/"))
	t = append(t, fmt.Sprintf("%.2f", cs.CpuPercentage))
	t = append(t, fmt.Sprintf("%.2f", cs.Memory))
	t = append(t, fmt.Sprintf("%.2f", cs.MemoryLimit))
	t = append(t, fmt.Sprintf("%.2f", cs.MemoryPercentage))

	return t
}

func Header() []string {
	return []string{"os", "id", "name", "cpup", "musage", "mlimit", "memp"}
}

func main() {
	// TODO: Add docker-endpoint param
	// TODO: Add sleep interval param
	// TODO: Add formatting param
	quit := make(chan error)
	done := make(chan string)
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
		go func(index int) {
			err := getDockerContainerStats(context.Background(), cli, stat, dockerContainerList[index])
			if err != nil {
				done <- dockerContainerList[index].ID
			}
		}(i)
	}

	writer := csv.NewWriter(os.Stdout)
	writer.Write(Header())
	writer.Flush()

	for {
		select {
		case s := <-stat:
			cs := CalculatedStats{
				OS:               s.os,
				ID:               s.container.ID,
				Name:             s.container.Names[0],
				CpuPercentage:    CalculateCPUPercentage(s.os, s.stats),
				Memory:           CalculateMemoryUsage(s.os, s.stats),
				MemoryLimit:      CalculateMemoryLimit(s.os, s.stats),
				MemoryPercentage: CalculateMemoryPercentage(s.os, s.stats),
			}
			err := writer.Write(cs.Strings())
			writer.Flush()

			if err != nil {
				fmt.Println(err)
			}
			//fmt.Printf("%s, %s, %s, cpu: %.2f mem: %.2f, mib: %s, limit: %s\n", s.os, cs.ID[:10], cs.Name, cs.CpuPercentage, cs.MemoryPercentage, units.BytesSize(cs.Memory), units.BytesSize(cs.MemoryLimit))

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
