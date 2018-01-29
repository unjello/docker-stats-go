package main

import (
	"github.com/docker/docker/api/types"
)

// based on docker client code:
// https://github.com/docker/cli/blob/4586609f71ac10dc43c846751c76e51a9ab81d56/cli/command/container/stats_helpers.go
func CalculateCPUPercentage(os string, v types.Stats) float64 {
	var cpuPercent = 0.0

	switch os {
	case "windows":
		var (
			possIntervals = uint64(v.Read.Sub(v.PreRead).Nanoseconds() / 100)
			intervalsUsed = v.CPUStats.CPUUsage.TotalUsage - v.PreCPUStats.CPUUsage.TotalUsage
		)
		possIntervals *= uint64(v.NumProcs)

		if possIntervals > 0 {
			cpuPercent = float64(intervalsUsed) / float64(possIntervals) * 100.0
		}
	case "linux":
		var (
			cpuDelta    = float64(v.CPUStats.CPUUsage.TotalUsage - v.PreCPUStats.CPUUsage.TotalUsage)
			systemDelta = float64(v.CPUStats.SystemUsage - v.PreCPUStats.SystemUsage)
			onlineCPUs  = float64(v.CPUStats.OnlineCPUs)
		)
		if onlineCPUs == 0.0 {
			onlineCPUs = float64(len(v.CPUStats.CPUUsage.PercpuUsage))
		}
		if systemDelta > 0.0 && cpuDelta > 0.0 {
			cpuPercent = (cpuDelta / systemDelta) * onlineCPUs * 100.0
		}
	}

	return cpuPercent
}

func CalculateMemoryUsage(os string, v types.Stats) float64 {
	var memoryUsage = 0.0

	switch os {
	case "windows":
		memoryUsage = float64(v.MemoryStats.PrivateWorkingSet)

	case "linux":
		memoryUsage = float64(v.MemoryStats.Usage - v.MemoryStats.Stats["cache"])
	}

	return memoryUsage
}

func CalculateMemoryLimit(os string, v types.Stats) float64 {
	var memoryLimit = 0.0

	switch os {
	case "linux":
		memoryLimit = float64(v.MemoryStats.Limit)
	}

	return memoryLimit
}

func CalculateMemoryPercentage(os string, v types.Stats) float64 {
	var (
		memoryUsage = CalculateMemoryUsage(os, v)
		memoryLimit = CalculateMemoryLimit(os, v)
	)

	if memoryLimit != 0 {
		return memoryUsage / memoryLimit * 100.0
	}

	return 0.0
}
