/// docker-stats-dump by andrzej lichnerowicz, unlicensed (~public domain)
/// program uses API v1.33 https://docs.docker.com/engine/api/v1.33/
package main

import (
	"bufio"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/go-units"
	flag "github.com/spf13/pflag"
)

func getDockerContainerStats(context context.Context, client *client.Client, stat chan<- Stats, container types.Container) error {
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
	Image            string
	CpuPercentage    float64
	Memory           float64
	MemoryLimit      float64
	MemoryPercentage float64
}

func (cs *CalculatedStats) Strings(isHumanReadable bool) []string {
	var t []string

	t = append(t, cs.OS)
	t = append(t, cs.ID[:10])
	t = append(t, strings.TrimLeft(cs.Name, "/"))
	t = append(t, cs.Image)
	if isHumanReadable {
		t = append(t, fmt.Sprintf("%.2f%%", cs.CpuPercentage))
		t = append(t, units.BytesSize(cs.Memory))
		t = append(t, units.BytesSize(cs.MemoryLimit))
		t = append(t, fmt.Sprintf("%.2f%%", cs.MemoryPercentage))
	} else {
		t = append(t, fmt.Sprintf("%.2f", cs.CpuPercentage))
		t = append(t, fmt.Sprintf("%.2f", cs.Memory))
		t = append(t, fmt.Sprintf("%.2f", cs.MemoryLimit))
		t = append(t, fmt.Sprintf("%.2f", cs.MemoryPercentage))
	}

	return t
}

func Header() []string {
	return []string{"os", "id", "name", "image", "cpup", "musage", "mlimit", "memp"}
}

type Options struct {
	IsHumanReadable bool
	Format          string
}

func (o *Options) Init() {
	flag.BoolVarP(&o.IsHumanReadable, "human-readable", "h", false, "output size numbers in IEC format")
	flag.StringVarP(&o.Format, "format", "f", "table", "format for results: table (default), csv, json")
}

func (o *Options) Parse() {
	flag.Parse()

	switch o.Format {
	case "table":
	case "csv":
	case "json":
	default:
		flag.PrintDefaults()
		os.Exit(0)
	}
}

type Writer interface {
	Write(record CalculatedStats, isHumanReadable bool) error
	WriteS(record []string) error
	Flush()
}

type TableWriter struct {
	w io.Writer
}

func NewTableWriter(w io.Writer) *TableWriter {
	return &TableWriter{w}
}

func (w *TableWriter) Write(record CalculatedStats, isHumanReadable bool) error {
	return w.WriteS(record.Strings(isHumanReadable))
}

func (w *TableWriter) WriteS(record []string) error {
	_, err := w.w.Write([]byte(fmt.Sprintln(strings.Join(record, "\t"))))
	return err
}

func (w *TableWriter) Flush() {}

type JsonWriter struct {
	w *json.Encoder
}

func NewJsonWriter(w *json.Encoder) *JsonWriter {
	return &JsonWriter{w}
}

func (w *JsonWriter) Write(record CalculatedStats, isHumanReadable bool) error {
	return w.w.Encode(record)
}
func (w *JsonWriter) WriteS(record []string) error {
	return nil
}
func (w *JsonWriter) Flush() {}

type CsvWriter struct {
	w *csv.Writer
}

func NewCsvWriter(w *csv.Writer) *CsvWriter {
	return &CsvWriter{w}
}

func (w *CsvWriter) Write(record CalculatedStats, isHumanReadable bool) error {
	return w.WriteS(record.Strings(isHumanReadable))
}

func (w *CsvWriter) WriteS(record []string) error {
	return w.w.Write(record)
}

func (w *CsvWriter) Flush() {
	w.w.Flush()
}

func main() {
	var options Options
	options.Init()
	options.Parse()

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
		panic(err)
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

	var writer Writer
	switch options.Format {
	case "csv":
		writer = NewCsvWriter(csv.NewWriter(os.Stdout))
	case "json":
		writer = NewJsonWriter(json.NewEncoder(os.Stdout))
	default:
		writer = NewTableWriter(os.Stdout)
	}

	writer.WriteS(Header())
	writer.Flush()

	for {
		select {
		case s := <-stat:
			cs := CalculatedStats{
				OS:               s.os,
				ID:               s.container.ID,
				Name:             s.container.Names[0],
				Image:            s.container.Image,
				CpuPercentage:    CalculateCPUPercentage(s.os, s.stats),
				Memory:           CalculateMemoryUsage(s.os, s.stats),
				MemoryLimit:      CalculateMemoryLimit(s.os, s.stats),
				MemoryPercentage: CalculateMemoryPercentage(s.os, s.stats),
			}
			err := writer.Write(cs, options.IsHumanReadable)
			if err != nil {
				panic(err)
			}
			writer.Flush()
		case <-done:
			dockerMonitors--
			if dockerMonitors == 0 {
				go func() {
					quit <- fmt.Errorf("No monitors left")
				}()
			}
		case <-quit:
			os.Exit(0)
		}
	}
}
