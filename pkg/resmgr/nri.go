package resmgr

import (
	"context"
	"fmt"
	"numaadj.huawei.com/pkg/typedef"
	"os"
	"sync"

	"github.com/containerd/nri/pkg/api"
	"github.com/containerd/nri/pkg/stub"
	"github.com/containerd/otelttrpc"
	"github.com/containerd/ttrpc"
	"k8s.io/klog/v2"
	"numaadj.huawei.com/pkg/apis/resaware/v1alpha1"
)

type nriPlugin struct {
	stub        stub.Stub
	resmgr      *resmgr
	cacheMu     sync.Mutex
	podCacheIns *typedef.PodCache
}

func newNRIPlugin(resmgr *resmgr) (*nriPlugin, error) {
	p := &nriPlugin{
		resmgr:      resmgr,
		podCacheIns: typedef.PodCacheInstance(),
		cacheMu:     sync.Mutex{},
	}

	klog.Infof("creating plugin...")

	return p, nil
}

func (p *nriPlugin) createStub() error {
	var (
		opts = []stub.Option{
			stub.WithPluginName(opt.NriPluginName),
			stub.WithPluginIdx(opt.NriPluginIdx),
			stub.WithSocketPath(opt.NriSocket),
			stub.WithOnClose(p.onClose),
			stub.WithTTRPCOptions(
				[]ttrpc.ClientOpts{
					ttrpc.WithUnaryClientInterceptor(
						otelttrpc.UnaryClientInterceptor(),
					),
				},
				[]ttrpc.ServerOpt{
					ttrpc.WithUnaryServerInterceptor(
						otelttrpc.UnaryServerInterceptor(),
					),
				},
			),
		}
		err error
	)

	klog.Info("creating plugin stub")

	if p.stub, err = stub.New(p, opts...); err != nil {
		return fmt.Errorf("failed to create NRI plugin stub: %w", err)
	}

	return nil
}

func (p *nriPlugin) start() error {
	if p == nil {
		return nil
	}
	klog.Info("starting plugin...")

	if err := p.createStub(); err != nil {
		return err
	}

	if err := p.stub.Start(context.Background()); err != nil {
		return fmt.Errorf("failed to start NRI plugin: %w", err)
	}
	return nil
}

func (p *nriPlugin) stop() {
	if p == nil {
		return
	}
	klog.Info("stop plugin...")
	p.stub.Stop()
}

func (p *nriPlugin) onClose() {
	klog.Error("connection to NRI/runtime lost, exiting...")
	os.Exit(1)
}

func (p *nriPlugin) Configure(ctx context.Context, cfg, runtime, version string) (stub.EventMask, error) {
	return api.MustParseEventMask("RunPodSandbox,StopPodSandbox"), nil
}

func (p *nriPlugin) RunPodSandbox(ctx context.Context, pod *api.PodSandbox) error {
	p.cacheMu.Lock()
	defer p.cacheMu.Unlock()

	if _, ok := pod.Labels[typedef.GroupLabel]; !ok {
		return nil
	}
	
	p.podCacheIns.UpdatePod(&typedef.PodInfo{
		Name:      pod.Name,
		UID:       pod.Uid,
		Namespace: pod.Namespace,
		Labels:    pod.Labels,
	})
	klog.Infof("pod %s/%s (id=%s) created, cached", pod.Namespace, pod.Name, pod.Uid)
	return nil
}

func (p *nriPlugin) StopPodSandbox(ctx context.Context, pod *api.PodSandbox) error {
	p.cacheMu.Lock()
	defer p.cacheMu.Unlock()
	p.podCacheIns.DelPod(pod.Uid)
	return nil
}

func (p *nriPlugin) updateContainers(oenuma *v1alpha1.Oenuma) error {
	_ = oenuma

	// for _, node := range oenuma.Spec.Node {
	// 	for _, podAfi := range node.PodAffinity {

	// 	}
	// }

	return nil
}

func (p *nriPlugin) getAdjustmentContainer() {

}
