package watcher

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	pb "exascale-metric-collector/pkg/client/grpc"
	pba "exascale-metric-collector/pkg/client/grpc/analysis"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

type Watcher struct {
	NodeIPMapper map[string]string
}

type server struct {
	pba.UnimplementedMetricCollectorServer
}

func InitWatcher() *Watcher {
	config, err := rest.InClusterConfig()

	if err != nil {
		fmt.Println(err.Error())
		config, err = clientcmd.BuildConfigFromFlags("", "/root/workspace/metric-collector/config/config")

		if err != nil {
			fmt.Println(err.Error())
			return nil
		}
	}

	clientset, _ := kubernetes.NewForConfig(config)
	podPrefix := clientset.CoreV1().Pods("keti-system")
	labelMap := make(map[string]string)
	labelMap["name"] = "keti-metric-collector"

	options := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(labelMap).String(),
	}
	metricPods, err := podPrefix.List(context.Background(), options)

	if err != nil {
		fmt.Println(err)
	}

	podIPMap := make(map[string]string)

	for _, pod := range metricPods.Items {
		podIPMap[pod.Spec.NodeName] = pod.Status.PodIP
	}

	return &Watcher{
		NodeIPMapper: podIPMap,
	}
}

func (w *Watcher) StartGRPCServer() {
	lis, err := net.Listen("tcp", ":9323")
	if err != nil {
		klog.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pba.RegisterMetricCollectorServer(s, &server{})
	fmt.Println("-----:: Metric Collector GRPC Server Running... ::-----")
	if err := s.Serve(lis); err != nil {
		klog.Fatalf("failed to serve: %v", err)
	}

}

func (w *Watcher) GetMultiMetric(ctx context.Context, r *pba.Request) (*pba.MultiMetric, error) {
	fmt.Println("-----:: Receive request ::-----")

	result := &pba.MultiMetric{
		NodeMetrics: make(map[string]*pba.NodeMetric),
	}

	for nodeName, podIP := range w.NodeIPMapper {
		fmt.Println("Node Name", nodeName)
		if strings.Contains(nodeName, "gpu") {
			respGPU := w.GetGPUMetric(podIP)
			if respGPU != nil {
				node_memory := respGPU.Message["Host_GPU_Memory_Gauge"].Metric[0].GetGauge().GetValue()
				gpu_power := respGPU.Message["Host_GPU_Power_Gauge"].Metric[0].GetGauge().GetValue()

				nm := &pba.NodeMetric{
					GpuMetrics: make(map[string]*pba.GPUMetric),
					NodeMemory: 0,
				}

				gm := &pba.GPUMetric{
					GpuPower: 0,
				}

				gm.GpuPower = float32(gpu_power)
				nm.GpuMetrics["uuid"] = gm
				nm.NodeMemory = float32(node_memory)
				result.NodeMetrics[nodeName] = nm

			}
		}
	}

	fmt.Println(result)

	return result, nil
}

func (w *Watcher) Work() {
	fmt.Println("Watcher start")
	for {
		fmt.Println("---------:: KETI-Metric Status ::---------")
		for nodeName, podIP := range w.NodeIPMapper {
			fmt.Println("Node Name", nodeName)
			if strings.Contains(nodeName, "gpu") {
				respGPU := w.GetGPUMetric(podIP)
				if respGPU != nil {
					str_gpuArch := w.ConvertGPUArchtoString(respGPU.Message["GPU_Architecture"].Metric[0].GetGauge().GetValue())
					fmt.Println("[Metric #01] Host GPU Core :", respGPU.Message["Host_GPU_Core_Gauge"].Metric[0].GetGauge().GetValue())
					fmt.Println("[Metric #02] Host GPU Memory :", respGPU.Message["Host_GPU_Memory_Gauge"].Metric[0].GetGauge().GetValue())
					fmt.Println("[Metric #03] Host GPU Power :", respGPU.Message["Host_GPU_Power_Gauge"].Metric[0].GetGauge().GetValue())
					fmt.Println("[Metric #04] GPU Memory Gauge:", respGPU.Message["GPU_Memory_Gauge"].Metric[0].GetGauge().GetValue())
					fmt.Println("[Metric #05] GPU Memory Counter :", respGPU.Message["GPU_Memory_Counter"].Metric[0].GetGauge().GetValue())
					fmt.Println("[Metric #06] GPU Temperature :", respGPU.Message["GPU_Temperature"].Metric[0].GetGauge().GetValue())
					fmt.Println("[Metric #07] GPU FLOPS :", respGPU.Message["GPU_Flops"].Metric[0].GetGauge().GetValue())
					fmt.Println("[Metric #08] GPU Fanspeed :", respGPU.Message["GPU_FanSpeed"].Metric[0].GetGauge().GetValue())
					fmt.Println("[Metric #09] GPU PodNum :", respGPU.Message["GPU_Pod_Num"].Metric[0].GetGauge().GetValue())
					fmt.Println("[Metric #10] GPU Architecture :", str_gpuArch)
					fmt.Println("[Metric #11] GPU Bandwidth :", respGPU.Message["GPU_Bandwidth"].Metric[0].GetGauge().GetValue())
					fmt.Println("[Metric #12] GPU Driver version :", respGPU.Message["GPU_Driver_Version"].Metric[0].Label[3].GetValue())
				}
			} else {
				fmt.Println("This node does not have GPU")
			}
			respNode := w.GetNodeMetric(podIP)
			if respNode != nil {
				fmt.Println("[Metric #13] Host CPU Core :", respNode.Message["CPU_Core_Gauge"].Metric[0].GetGauge().GetValue())
				fmt.Println("[Metric #14] Host Memory :", respNode.Message["Memory_Gauge"].Metric[0].GetGauge().GetValue())
				fmt.Println("[Metric #15] Host Storage :", respNode.Message["Storage_Gauge"].Metric[0].GetGauge().GetValue())
				fmt.Println("[Metric #16] CPU Core :", respNode.Message["CPU_Core_Counter"].Metric[0].GetGauge().GetValue())
				fmt.Println("[Metric #17] Memory :", respNode.Message["Memory_Counter"].Metric[0].GetGauge().GetValue())
				fmt.Println("[Metric #18] Storage :", respNode.Message["Storage_Counter"].Metric[0].GetGauge().GetValue())
				fmt.Println("[Metric #19] NetworkRX:", respNode.Message["Network_Gauge"].Metric[0].GetGauge().GetValue())
				fmt.Println("[Metric #20] NetworkTX :", respNode.Message["Network_Counter"].Metric[0].GetGauge().GetValue())
			}
		}
		time.Sleep(time.Second * 10)
	}
}

func (w *Watcher) GetGPUMetric(podIP string) *pb.Response {
	host := podIP + ":50052"
	conn, err := grpc.Dial(host, grpc.WithTransportCredentials(insecure.NewCredentials()))

	if err != nil {
		klog.Errorln("did not connect: %v \n", err)
	}

	defer conn.Close()

	metricClient := pb.NewMetricGathererClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	//fmt.Println(metricClient)

	r, err := metricClient.GetGPU(ctx, &pb.Request{})

	if err != nil {
		klog.Errorf("could not request: %v \n", err)

		fmt.Println(r)
	}

	return r
}

func (w *Watcher) GetNodeMetric(podIP string) *pb.Response {
	host := podIP + ":50052"
	conn, err := grpc.Dial(host, grpc.WithTransportCredentials(insecure.NewCredentials()))

	if err != nil {
		klog.Errorln("did not connect: %v \n", err)
	}

	defer conn.Close()

	metricClient := pb.NewMetricGathererClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	//fmt.Println(metricClient)

	r, err := metricClient.GetNode(ctx, &pb.Request{})

	if err != nil {
		klog.Errorf("could not request: %v \n", err)

		r = nil
	}

	return r
}

func (w *Watcher) ConvertGPUArchtoString(gpuarch float64) string {

	/*
		#define NVML_DEVICE_ARCH_KEPLER    2 // Devices based on the NVIDIA Kepler architecture
		#define NVML_DEVICE_ARCH_MAXWELL   3 // Devices based on the NVIDIA Maxwell architecture
		#define NVML_DEVICE_ARCH_PASCAL    4 // Devices based on the NVIDIA Pascal architecture
		#define NVML_DEVICE_ARCH_VOLTA     5 // Devices based on the NVIDIA Volta architecture
		#define NVML_DEVICE_ARCH_TURING    6 // Devices based on the NVIDIA Turing architecture
		#define NVML_DEVICE_ARCH_AMPERE    7 // Devices based on the NVIDIA Ampere architecture
	*/

	if gpuarch == 2 {
		return "KEPLER"
	}

	if gpuarch == 3 {
		return "MAXWELL"
	}

	if gpuarch == 4 {
		return "PASCAL"
	}

	if gpuarch == 5 {
		return "VOLTA"
	}

	if gpuarch == 6 {
		return "TURING"
	}

	if gpuarch == 7 {
		return "AMPERE"
	}

	return ""
}
