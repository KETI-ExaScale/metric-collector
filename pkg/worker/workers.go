package worker

import (
	"context"
	"exascale-metric-collector/pkg/client/kubelet"
	"exascale-metric-collector/pkg/decode"
	"exascale-metric-collector/pkg/storage"
	"fmt"
	"net"
	"os"

	"github.com/NVIDIA/go-nvml/pkg/nvml"
	"github.com/prometheus/client_golang/prometheus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

type MetricWorker struct {
	KETIGPURegistry  *prometheus.Registry
	KETINodeRegistry *prometheus.Registry
	KubeletClient    *kubelet.KubeletClient
}

func Initmetrics(metricType bool, nodeName string, podName string) *MetricWorker {
	// Since we are dealing with custom Collector implementations, it might
	// be a good idea to try it out with a pedantic registry.
	fmt.Println("Initializing metrics...")

	worker := &MetricWorker{
		KETIGPURegistry:  prometheus.NewRegistry(),
		KETINodeRegistry: prometheus.NewRegistry(),
	}

	//reg := prometheus.NewPedanticRegistry()
	config, err := rest.InClusterConfig()
	if err != nil {
		fmt.Println(err.Error())
		config, err = clientcmd.BuildConfigFromFlags("", "/root/workspace/metric-collector/config/config")
		if err != nil {
			fmt.Println(err.Error())
			return nil
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}
	node, err := clientset.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("Error: %v\n", err)
	}

	for _, addr := range node.Status.Addresses {
		if addr.Type == "InternalIP" {
			worker.KubeletClient = kubelet.NewKubeletClient(addr.Address, config.BearerToken)
			break
		}
	}
	NewClusterManager(metricType, worker.KETINodeRegistry, worker.KETIGPURegistry, clientset, worker.KubeletClient, podName)
	return worker
}

type ClusterManager struct {
	MetricType string
}
type GPUCollector struct {
	MapperHost         string
	ClusterManager     *ClusterManager
	HostGPUCoreGauge   *prometheus.Desc
	HostGPUMemoryGauge *prometheus.Desc
	HostGPUPowerGauge  *prometheus.Desc
	//@ria
	GPUMemoryGauge   *prometheus.Desc
	GPUMemoryCounter *prometheus.Desc
	GPUTemperature   *prometheus.Desc
	GPUFlops         *prometheus.Desc
	GPUArchitecture  *prometheus.Desc
	GPUPodNum        *prometheus.Desc
	GPUFanspeed      *prometheus.Desc
	GPUBandwidth     *prometheus.Desc
	GPUDriverVersion *prometheus.Desc
	ClientSet        *kubernetes.Clientset
	KubeletClient    *kubelet.KubeletClient
}
type NodeCollector struct {
	ClusterManager   *ClusterManager
	CPUCoreGauge     *prometheus.Desc
	CPUCoreCounter   *prometheus.Desc
	MemoryGauge      *prometheus.Desc
	MemoryCounter    *prometheus.Desc
	StorageGauge     *prometheus.Desc
	StorageCounter   *prometheus.Desc
	NetworkRXCounter *prometheus.Desc
	NetworkTXCounter *prometheus.Desc
	ClientSet        *kubernetes.Clientset
	KubeletClient    *kubelet.KubeletClient
}

func NewClusterManager(metricType bool, Nodereg prometheus.Registerer, GPUreg prometheus.Registerer,
	clientset *kubernetes.Clientset, kubeletClient *kubelet.KubeletClient, podName string) {
	mapperPod, err := clientset.CoreV1().Pods("keti-system").Get(context.Background(), podName, metav1.GetOptions{})
	if err != nil {
		klog.Errorln(err)
	}

	if metricType {
		gcm := &ClusterManager{
			MetricType: "GPUMetric",
		}
		gc := GPUCollector{
			MapperHost:     net.JoinHostPort(mapperPod.Status.PodIP, "50052"),
			KubeletClient:  kubeletClient,
			ClusterManager: gcm,
			HostGPUCoreGauge: prometheus.NewDesc(
				"Host_GPU_Core_Gauge",
				"Host GPU Core Utilization with Percent",
				[]string{"devicename", "deviceuuid"}, nil,
			),
			HostGPUMemoryGauge: prometheus.NewDesc(
				"Host_GPU_Memory_Gauge",
				"Host GPU Memory Utilization with Percent",
				[]string{"devicename", "deviceuuid"}, nil,
			),
			HostGPUPowerGauge: prometheus.NewDesc(
				"Host_GPU_Power_Gauge",
				"Host GPU Power Utilization with Percent",
				[]string{"devicename", "deviceuuid"}, nil,
			),
			//@ria
			GPUMemoryGauge: prometheus.NewDesc(
				"GPU_Memory_Gauge",
				"Pod GPU Memory Utilization with Percent",
				[]string{"devicename", "deviceuuid"}, nil,
			),
			GPUMemoryCounter: prometheus.NewDesc(
				"GPU_Memory_Counter",
				"Pod GPU Memory Utilization with Counter",
				[]string{"devicename", "deviceuuid"}, nil,
			),
			GPUTemperature: prometheus.NewDesc(
				"GPU_Temperature",
				"GPU Temprature",
				[]string{"devicename", "deviceuuid"}, nil,
			),
			GPUFlops: prometheus.NewDesc(
				"GPU_Flops",
				"GPU Flops",
				[]string{"devicename", "deviceuuid"}, nil,
			),
			GPUArchitecture: prometheus.NewDesc(
				"GPU_Architecture",
				"GPU Architecture",
				[]string{"devicename", "deviceuuid"}, nil,
			),
			GPUPodNum: prometheus.NewDesc(
				"GPU_Pod_Num",
				"Pod Count Use GPU",
				[]string{"devicename", "deviceuuid"}, nil),
			GPUFanspeed: prometheus.NewDesc(
				"GPU_FanSpeed",
				"GPU_FanSpeed",
				[]string{"devicename", "deviceuuid"}, nil,
			),
			GPUBandwidth: prometheus.NewDesc(
				"GPU_Bandwidth",
				"GPU Bandwidth",
				[]string{"devicename", "deviceuuid"}, nil,
			),
			GPUDriverVersion: prometheus.NewDesc(
				"GPU_Driver_Version",
				"GPU Driver Version",
				[]string{"devicename", "deviceuuid", "version"}, nil,
			),
			ClientSet: clientset,
		}
		prometheus.WrapRegistererWith(prometheus.Labels{"metricType": gcm.MetricType}, GPUreg).MustRegister(gc)
	}
	ncm := &ClusterManager{
		MetricType: "NodeMetric",
	}
	nc := NodeCollector{
		ClusterManager: ncm,
		KubeletClient:  kubeletClient,
		CPUCoreGauge: prometheus.NewDesc(
			"CPU_Core_Gauge",
			"Host CPU Utilization with Percent",
			[]string{"clustername", "podnamespace", "podname"}, nil,
		),
		CPUCoreCounter: prometheus.NewDesc(
			"CPU_Core_Counter",
			"Pod CPU Utilization with Counter",
			[]string{"clustername", "podnamespace", "podname"}, nil,
		),
		MemoryGauge: prometheus.NewDesc(
			"Memory_Gauge",
			"Host Memory Utilization with Percent",
			[]string{"clustername", "podnamespace", "podname"}, nil,
		),
		MemoryCounter: prometheus.NewDesc(
			"Memory_Counter",
			"Pod Memory Utilization with Counter",
			[]string{"clustername", "podnamespace", "podname"}, nil,
		),
		StorageGauge: prometheus.NewDesc(
			"Storage_Gauge",
			"Host Storage Utilization with Percent",
			[]string{"clustername", "podnamespace", "podname"}, nil,
		),
		StorageCounter: prometheus.NewDesc(
			"Storage_Counter",
			"Pod Storage Utilization with Counter",
			[]string{"clustername", "podnamespace", "podname"}, nil,
		),
		NetworkRXCounter: prometheus.NewDesc(
			"Network_Gauge",
			"Pod Network RX Utilization with Counter",
			[]string{"clustername", "podnamespace", "podname"}, nil,
		),
		NetworkTXCounter: prometheus.NewDesc(
			"Network_Counter",
			"Pod Network TX Utilization with Counter",
			[]string{"clustername", "podnamespace", "podname"}, nil,
		),
		ClientSet: clientset,
	}
	prometheus.WrapRegistererWith(prometheus.Labels{"metricType": ncm.MetricType}, Nodereg).MustRegister(nc)
}

func (nc NodeCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- nc.CPUCoreCounter
	ch <- nc.CPUCoreGauge
	ch <- nc.MemoryCounter
	ch <- nc.MemoryGauge
	ch <- nc.NetworkRXCounter
	ch <- nc.NetworkTXCounter
	ch <- nc.StorageCounter
	ch <- nc.StorageGauge
}
func (nc NodeCollector) Collect(ch chan<- prometheus.Metric) {
	nodeName := os.Getenv("NODE_NAME")
	node, err := nc.ClientSet.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
	if err != nil {
		klog.Errorln(err)
	}

	totalCPUQuantity := node.Status.Allocatable["cpu"]
	totalCPU, _ := totalCPUQuantity.AsInt64()
	totalMemoryQuantity := node.Status.Allocatable["memory"]
	totalMemory, _ := totalMemoryQuantity.AsInt64()
	totalStorageQuantity := node.Status.Allocatable["ephemeral-storage"]
	totalStorage, _ := totalStorageQuantity.AsInt64()

	collection, err := Scrap(nc.KubeletClient, node)
	if err != nil {
		klog.Errorln(err)
	}
	clusterName := collection.ClusterName

	for _, podMetric := range collection.Metricsbatch.Pods {
		cpuUsage, _ := podMetric.CPUUsageNanoCores.AsInt64()
		cpuPercent := float64(cpuUsage) / float64(totalCPU)
		memoryUsage, _ := podMetric.MemoryUsageBytes.AsInt64()
		memoryPercent := float64(memoryUsage) / float64(totalMemory)
		storageUsage, _ := podMetric.FsUsedBytes.AsInt64()
		storagePercent := float64(storageUsage) / float64(totalStorage)

		networkrxUsage, _ := podMetric.NetworkRxBytes.AsInt64()
		networktxUsage, _ := podMetric.NetworkTxBytes.AsInt64()

		ch <- prometheus.MustNewConstMetric(
			nc.CPUCoreGauge,
			prometheus.GaugeValue,
			float64(cpuPercent*100),
			clusterName, podMetric.Namespace, podMetric.Name,
		)
		ch <- prometheus.MustNewConstMetric(
			nc.CPUCoreCounter,
			prometheus.GaugeValue,
			float64(cpuUsage),
			clusterName, podMetric.Namespace, podMetric.Name,
		)
		ch <- prometheus.MustNewConstMetric(
			nc.MemoryCounter,
			prometheus.GaugeValue,
			float64(memoryUsage),
			clusterName, podMetric.Namespace, podMetric.Name,
		)
		ch <- prometheus.MustNewConstMetric(
			nc.MemoryGauge,
			prometheus.GaugeValue,
			float64(memoryPercent*100),
			clusterName, podMetric.Namespace, podMetric.Name,
		)
		ch <- prometheus.MustNewConstMetric(
			nc.NetworkRXCounter,
			prometheus.GaugeValue,
			float64(networkrxUsage),
			clusterName, podMetric.Namespace, podMetric.Name,
		)
		ch <- prometheus.MustNewConstMetric(
			nc.NetworkTXCounter,
			prometheus.GaugeValue,
			float64(networktxUsage),
			clusterName, podMetric.Namespace, podMetric.Name,
		)
		ch <- prometheus.MustNewConstMetric(
			nc.StorageCounter,
			prometheus.GaugeValue,
			float64(storageUsage),
			clusterName, podMetric.Namespace, podMetric.Name,
		)
		ch <- prometheus.MustNewConstMetric(
			nc.StorageGauge,
			prometheus.GaugeValue,
			float64(storagePercent*100),
			clusterName, podMetric.Namespace, podMetric.Name,
		)
	}

}

func (gc GPUCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- gc.HostGPUCoreGauge   //
	ch <- gc.HostGPUMemoryGauge //
	ch <- gc.HostGPUPowerGauge  //
	ch <- gc.GPUArchitecture    //
	ch <- gc.GPUBandwidth       //
	ch <- gc.GPUDriverVersion   //
	ch <- gc.GPUFlops           //
	//@ria
	ch <- gc.GPUMemoryCounter
	ch <- gc.GPUMemoryGauge
	ch <- gc.GPUPodNum
	ch <- gc.GPUTemperature //
	ch <- gc.GPUFanspeed    //
}
func (gc GPUCollector) Collect(ch chan<- prometheus.Metric) {
	// conn, err := grpc.Dial(gc.MapperHost, grpc.WithInsecure())
	// if err != nil {
	// 	log.Fatalf("Failed to dial: %v", err)
	// }
	// defer conn.Close()
	// nodeName := os.Getenv("NODE_NAME")
	// client := podmap.NewPodMapperClient(conn)
	// req := &podmap.Request{}
	// res, err := client.PodPID(context.Background(), req)
	// if err != nil {
	// 	klog.Fatalf("Failed to call YourMethod: %v", err)
	// }
	// containerRes, err := client.PodContainer(context.Background(), req)
	// if err != nil {
	// 	klog.Fatalf("Failed to call YourMethod: %v", err)
	// }
	// PIDMap := res.Message
	// ContainerMap := containerRes.Message
	// klog.Infoln(ContainerMap)
	procInfo := make(map[string]uint64)
	// podList, err := gc.ClientSet.CoreV1().Pods(v1.NamespaceAll).List(context.Background(), metav1.ListOptions{})
	// if err != nil {
	// 	klog.Errorln(err)
	// }

	nvmlReturn := nvml.Init()
	if nvmlReturn != nvml.SUCCESS {
		klog.Errorf("Unable to initialize NVML: %v\n", nvmlReturn)
	}
	defer func() {
		nvmlReturn := nvml.Shutdown()
		if nvmlReturn != nvml.SUCCESS {
			//log.Fatalf("Unable to shutdown NVML: %v", ret)
			fmt.Printf("Unable to shutdown NVML: %v\n", nvmlReturn)
		}
	}()
	driverVersion, nvmlReturn := nvml.SystemGetDriverVersion()
	if nvmlReturn != nvml.SUCCESS {
		fmt.Printf("Failed to get driver version\n")
		return
	}
	count, nvmlReturn := nvml.DeviceGetCount()
	if nvmlReturn != nvml.SUCCESS {
		//log.Fatalf("Unable to get device count: %v", ret)
		klog.Infof("Unable to get device count: %v", nvmlReturn)
		count = 0
	}

	for i := 0; i < count; i++ {
		device, ret := nvml.DeviceGetHandleByIndex(i)
		if ret != nvml.SUCCESS {
			klog.Fatalf("Unable to get device at index %d: %v", i, ret)
		}
		memoryInfo, ret := device.GetMemoryInfo()
		if ret != nvml.SUCCESS {
			klog.Fatalf("Unable to get memoryInfo: %v", ret)
		} else {
			klog.Infof("Memory Get Complete: %v\n", memoryInfo.Total)
		}

		maxMemory := memoryInfo.Total

		deviceName, ret := device.GetName()
		if ret != nvml.SUCCESS {
			klog.Fatalf("Unable to get deviceName: %v", ret)
		} else {
			klog.Infof("DeviceName Get Complete: %v\n", deviceName)
		}
		hostGPUUsage, ret := device.GetUtilizationRates()
		if ret != nvml.SUCCESS {
			klog.Fatalf("Unable to get hostGPUUsage: %v", ret)
		} else {
			klog.Infof("HostGPUUsage Get Complete: %v\n", hostGPUUsage.Memory)
		}
		hostGPUPower, ret := device.GetPowerUsage()
		if ret != nvml.SUCCESS {
			klog.Fatalf("Unable to get hostGPUPower: %v", ret)
		} else {
			klog.Infof("HostGPUPower Get Complete: %v\n", hostGPUPower)
		}
		deviceTemperature, ret := device.GetTemperature(0)
		if ret != nvml.SUCCESS {
			klog.Fatalf("Unable to get deviceTemperature: %v", ret)
		} else {
			klog.Infof("deviceTemperature Get Complete: %v\n", deviceTemperature)
		}
		deviceUUID, ret := device.GetUUID()
		if ret != nvml.SUCCESS {
			klog.Fatalf("Unable to get deviceUUID: %v", ret)
		} else {
			klog.Infof("deviceUUID Get Complete: %v\n", deviceTemperature)
		}

		computeCapabilityMajor, computeCapabilityMinor, ret := device.GetCudaComputeCapability()
		if ret != nvml.SUCCESS {
			klog.Fatalf("Unable to get computeCapabilityMajor: %v", ret)
		}
		architecture, ret := nvml.DeviceGetArchitecture(device)
		if ret != nvml.SUCCESS {
			klog.Fatalf("Unable to get architecture: %v", ret)
		}
		clock, ret := device.GetMaxClockInfo(nvml.CLOCK_SM)
		if ret != nvml.SUCCESS {
			klog.Fatalf("Unable to get clock: %v", ret)
		}
		multiprocessorCount, ret := device.GetNumGpuCores()
		if ret != nvml.SUCCESS {
			klog.Fatalf("Unable to get multiprocessorCount: %v", ret)
		}
		deviceFlops := 2 * float64(clock) * float64(multiprocessorCount) * float64(computeCapabilityMajor) * float64(computeCapabilityMinor)
		deviceBandwidth := getMemoryBandwidth(device)
		deviceFan, ret := device.GetFanSpeed()
		if ret != nvml.SUCCESS {
			klog.Fatalf("Unable to get deviceFan: %v", ret)
		}

		ch <- prometheus.MustNewConstMetric(
			gc.HostGPUCoreGauge,
			prometheus.GaugeValue,
			float64(hostGPUUsage.Gpu),
			deviceName, deviceUUID,
		)
		ch <- prometheus.MustNewConstMetric(
			gc.HostGPUMemoryGauge,
			prometheus.GaugeValue,
			float64(hostGPUUsage.Memory),
			deviceName, deviceUUID,
		)
		ch <- prometheus.MustNewConstMetric(
			gc.HostGPUPowerGauge,
			prometheus.GaugeValue,
			float64(hostGPUPower),
			deviceName, deviceUUID,
		)
		ch <- prometheus.MustNewConstMetric(
			gc.GPUTemperature,
			prometheus.GaugeValue,
			float64(deviceTemperature),
			deviceName, deviceUUID,
		)
		ch <- prometheus.MustNewConstMetric(
			gc.GPUFlops,
			prometheus.GaugeValue,
			float64(deviceFlops),
			deviceName, deviceUUID,
		)
		ch <- prometheus.MustNewConstMetric(
			gc.GPUArchitecture,
			prometheus.GaugeValue,
			float64(architecture),
			deviceName, deviceUUID,
		)
		ch <- prometheus.MustNewConstMetric(
			gc.GPUFanspeed,
			prometheus.GaugeValue,
			float64(deviceFan),
			deviceName, deviceUUID,
		)
		ch <- prometheus.MustNewConstMetric(
			gc.GPUBandwidth,
			prometheus.GaugeValue,
			float64(deviceBandwidth),
			deviceName, deviceUUID,
		)
		ch <- prometheus.MustNewConstMetric(
			gc.GPUDriverVersion,
			prometheus.GaugeValue,
			float64(0),
			deviceName, deviceUUID, driverVersion,
		)
		procInfo = GetGPUProcess(device)

		ch <- prometheus.MustNewConstMetric(
			gc.GPUPodNum,
			prometheus.GaugeValue,
			float64(len(procInfo)),
			deviceName, deviceUUID,
		)
		//@ria
		if len(procInfo) == 0 {
			ch <- prometheus.MustNewConstMetric(
				gc.GPUMemoryCounter,
				prometheus.GaugeValue,
				float64(0),
				deviceName, deviceUUID,
			)
			ch <- prometheus.MustNewConstMetric(
				gc.GPUMemoryGauge,
				prometheus.GaugeValue,
				0,
				deviceName, deviceUUID,
			)
		} else {
			for _, value := range procInfo {
				ch <- prometheus.MustNewConstMetric(
					gc.GPUMemoryCounter,
					prometheus.GaugeValue,
					float64(value),
					deviceName, deviceUUID,
				)
				ch <- prometheus.MustNewConstMetric(
					gc.GPUMemoryGauge,
					prometheus.GaugeValue,
					(float64(value)/float64(maxMemory))*100,
					deviceName, deviceUUID,
				)
			}
		}

		// for _, pod := range podList.Items {
		// 	if pod.Namespace == "kube-system" {
		// 		continue
		// 	}
		// 	if pod.Spec.NodeName != nodeName {
		// 		continue
		// 	}
		// 	if _, ok := procInfo[pid]; ok {
		// 		ch <- prometheus.MustNewConstMetric(
		// 			gc.GPUMemoryCounter,
		// 			prometheus.GaugeValue,
		// 			float64(procInfo[pid]),
		// 			pod.Namespace, pod.Name, deviceUUID,
		// 		)
		// 		ch <- prometheus.MustNewConstMetric(
		// 			gc.GPUMemoryGauge,
		// 			prometheus.GaugeValue,
		// 			float64(procInfo[pid])/float64(maxMemory)*100,
		// 			pod.Namespace, pod.Name, deviceUUID,
		// 		)
		// 	}
		// }
	}
}

func Scrap(kubelet_client *kubelet.KubeletClient, node *v1.Node) (*storage.Collection, error) {
	//fmt.Println("Func Scrap Called")

	//startTime := clock.MyClock.Now()

	metrics, err := CollectNode(kubelet_client, node)
	if err != nil {
		err = fmt.Errorf("unable to fully scrape metrics from node %s: %v", node.Name, err)
	}

	var errs []error
	res := &storage.Collection{
		Metricsbatch: metrics,
		ClusterName:  os.Getenv("CLUSTER_NAME"),
	}

	//fmt.Println("ScrapeMetrics: time: ", clock.MyClock.Since(startTime), "nodes: ", nodeNum, "pods: ", podNum)
	return res, utilerrors.NewAggregate(errs)
}

func CollectNode(kubelet_client *kubelet.KubeletClient, node *v1.Node) (*storage.MetricsBatch, error) {

	summary, err := kubelet_client.GetSummary()
	//fmt.Println("summary : ", summary)
	if err != nil {
		return nil, fmt.Errorf("unable to fetch metrics from Kubelet %s (%s): %v", node.Name, node.Status.Addresses[0].Address, err)
	}

	return decode.DecodeBatch(summary)
}

func GetGPUProcess(device nvml.Device) map[string]uint64 {
	nvmlProc, ret := device.GetComputeRunningProcesses()
	if ret != nvml.SUCCESS {
		klog.Fatalf("Unable to get nvmlProc: %v", ret)
	} else {
		klog.Infof("nvmlProc Get Complete: %v\n", nvmlProc)
	}
	nvmlmpsProc, ret := device.GetMPSComputeRunningProcesses()
	if ret != nvml.SUCCESS {
		klog.Fatalf("Unable to get nvmlmpsProc: %v", ret)
	} else {
		klog.Infof("nvmlmpsProc Get Complete: %v\n", nvmlmpsProc)
	}
	nvmlProc = append(nvmlProc, nvmlmpsProc...)

	procInfo := make(map[string]uint64)

	for _, proc := range nvmlProc {
		strPid := fmt.Sprint(proc.Pid)
		procInfo[strPid] = proc.UsedGpuMemory
	}
	return procInfo
}

func getMemoryBandwidth(device nvml.Device) float64 {
	busWidth, ret := device.GetMemoryBusWidth()
	if ret != nvml.SUCCESS {
		klog.Fatalf("Unable to get busWidth: %v", ret)
	}

	memClock, ret := device.GetMaxClockInfo(nvml.CLOCK_MEM)
	if ret != nvml.SUCCESS {
		klog.Fatalf("Unable to get memClock: %v", ret)
	}

	bandwidth := float64(busWidth) * float64(memClock) * 2 / 8 / 1e6 // GB/s
	return bandwidth
}
