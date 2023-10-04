package collector

import (
	"context"
	"encoding/json"

	"fmt"
	"net"

	metricCore "exascale-metric-collector/pkg/client/grpc"

	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"k8s.io/klog/v2"
)

type Provider struct {
	IsGPU bool
	metricCore.UnsafeMetricGathererServer
	PromRegistry *prometheus.Registry
}

func (p *Provider) Node(ctx context.Context, req *metricCore.Request) (*metricCore.Response, error) {
	metricFamily, err := p.PromRegistry.Gather()
	if err != nil {
		return nil, err
	}

	metricMap := make(map[string]*metricCore.MetricFamily)
	for _, mf := range metricFamily {
		tmp_mf := &metricCore.MetricFamily{}

		jsonByte, err := json.Marshal(mf)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(jsonByte, tmp_mf)
		if err != nil {
			return nil, err
		}

		metricMap[*mf.Name] = tmp_mf
	}
	return &metricCore.Response{Message: metricMap}, nil
}

func (p *Provider) GPU(ctx context.Context, req *metricCore.Request) (*metricCore.Response, error) {
	if !p.IsGPU {
		return nil, fmt.Errorf("this node does not have a GPU. Please plug in GPU or install GPU driver")
	}
	metricFamily, err := p.PromRegistry.Gather()
	if err != nil {
		return nil, err
	}
	metricMap := make(map[string]*metricCore.MetricFamily)
	for _, mf := range metricFamily {
		tmp_mf := &metricCore.MetricFamily{}

		jsonByte, err := json.Marshal(mf)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(jsonByte, tmp_mf)
		if err != nil {
			return nil, err
		}

		metricMap[*mf.Name] = tmp_mf
	}
	return &metricCore.Response{Message: metricMap}, nil
}

func RunCollectorServer(isGPU bool, reg *prometheus.Registry) {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		klog.Fatalf("failed to listen: %v", err)
	}
	nodeServer := grpc.NewServer()
	metricCore.RegisterMetricGathererServer(nodeServer, &Provider{IsGPU: isGPU, PromRegistry: reg})
	fmt.Println("node server started...")
	if err := nodeServer.Serve(lis); err != nil {
		klog.Fatalf("failed to serve: %v", err)
	}
}
