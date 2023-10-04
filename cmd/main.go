package main

import (
	"exascale-metric-collector/pkg/collector"
	"exascale-metric-collector/pkg/worker"
	"os"
	"strings"
)

func main() {
	run()
}

func run() {
	//isGPU := os.Getenv("IS_GPUNODE")
	nodeName := os.Getenv("NODE_NAME")

	if strings.Compare(nodeName, "gpu-node1") == 0 {
		workerReg := worker.Initmetrics(true, nodeName)
		go collector.RunCollectorServer(true, workerReg.KETIRegistry)
		workerReg.StartHTTPServer()
	} else if strings.Compare(nodeName, "gpu-node2") == 0 {
		workerReg := worker.Initmetrics(true, nodeName)
		go collector.RunCollectorServer(true, workerReg.KETIRegistry)
		workerReg.StartHTTPServer()
	} else {
		workerReg := worker.Initmetrics(false, nodeName)
		go collector.RunCollectorServer(false, workerReg.KETIRegistry)
		workerReg.StartHTTPServer()
	}
}
