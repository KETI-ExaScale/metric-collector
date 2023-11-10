package main

import (
	"exascale-metric-collector/pkg/collector"
	"exascale-metric-collector/pkg/watcher"
	"exascale-metric-collector/pkg/worker"

	"strings"
	"time"

	"os"
)

func main() {
	run()
}

func run() {
	//isGPU := os.Getenv("IS_GPUNODE")
	nodeName := os.Getenv("NODE_NAME")
	podName := os.Getenv("POD_NAME")

	if strings.Contains(nodeName, "gpu") {
		// fmt.Printf("nvml: %v\n", nvml.ErrorString(nvmlReturn))
		workerReg := worker.Initmetrics(true, nodeName, podName)
		collector.RunCollectorServer(true, workerReg.KETINodeRegistry, workerReg.KETIGPURegistry)
		// time.Sleep(time.Second * 10)
		// watcher := watcher.InitWatcher()
		// watcher.Work()
		// workerReg.StartHTTPServer()
	} else {
		workerReg := worker.Initmetrics(false, nodeName, podName)
		if strings.Contains(nodeName, "master") {
			go collector.RunCollectorServer(false, workerReg.KETINodeRegistry, workerReg.KETIGPURegistry)
			time.Sleep(time.Second * 10)
			watcher := watcher.InitWatcher()
			watcher.Work()
		} else {
			collector.RunCollectorServer(false, workerReg.KETINodeRegistry, workerReg.KETIGPURegistry)
		}
		// workerReg.StartHTTPServer()
	}

	// if strings.Compare(nodeName, "gpu-node1") == 0 {
	// 	workerReg := worker.Initmetrics(true, nodeName)
	// 	go collector.RunCollectorServer(true, workerReg.KETIRegistry)
	// 	workerReg.StartHTTPServer()
	// } else if strings.Compare(nodeName, "gpu-node2") == 0 {
	// 	workerReg := worker.Initmetrics(true, nodeName)
	// 	go collector.RunCollectorServer(true, workerReg.KETIRegistry)
	// 	workerReg.StartHTTPServer()
	// } else {
	// 	workerReg := worker.Initmetrics(false, nodeName)
	// 	go collector.RunCollectorServer(false, workerReg.KETIRegistry)
	// 	workerReg.StartHTTPServer()
	// }
}
