package resmgr

import (
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
	"k8s.io/klog/v2"
	"numaadj.huawei.com/pkg/agent"
	"numaadj.huawei.com/pkg/policy"
)

type ResourceManager interface {
	// Start starts the resource manager.
	Start() error

	// Stop stops the resource manager.
	Stop() error

	// SendEvent sends an evnet to be processed by the resource manager.
	SendEvent(event interface{}) error
}

var (
	log = logrus.New()
)

type resmgr struct {
	sync.RWMutex
	agent   *agent.Agent
	nri     *nriPlugin // NRI plugin, if we're running as such
	running bool
}

func NewResourceManager(agt *agent.Agent, backend interface{}) (*resmgr, error) {
	m := &resmgr{
		agent: agt,
	}

	klog.Info("running as an NRI plugin...")
	nrip, err := newNRIPlugin(m)
	if err != nil {
		return nil, err
	}
	m.nri = nrip

	// TODO: m set policy

	return m, nil
}

func (m *resmgr) Start() error {
	klog.Infof("starting agent, waiting for initial configuration...")

	return m.agent.Start(m.updateConfig)
}

func (m *resmgr) updateConfig(newCfg interface{}) error {
	if newCfg == nil {
		return fmt.Errorf("can not run without effective configuration...")
	}

	crdCfg, ok := newCfg.(*policy.Config)
	if !ok {
		return fmt.Errorf("can not run without effective configuration...")
	}

	if !m.running {
		klog.Infof("aquired initial configuration")

		if err := m.start(newCfg); err != nil {
			klog.Fatalf("failed to start with initial configuration: %v", err)
		}
		m.running = true
	}

	return m.reconfigCrd(crdCfg)
}

func (m *resmgr) reconfigCrd(crdCfg *policy.Config) error {
	if err := crdCfg.Validate(); err != nil {
		return fmt.Errorf("in validate crd configuration")
	}

	klog.Infof("update size: %v, crdCfg: %v", len(crdCfg.GetUpdate()), crdCfg.GetUpdate())

	//for _, upd := range crdCfg.GetUpdate() {
	//	klog.Warningf("containerdId: %s, cpuset: %s, memset: %s", upd.ContainerId, upd.Linux.Resources.Cpu.Cpus, upd.Linux.Resources.Cpu.Mems)
	//}

	_, err := m.nri.stub.UpdateContainers(crdCfg.GetUpdate())

	if err != nil {
		return fmt.Errorf("failed to adjust numa node by nriPlugin, %v", err)
	}
	return nil
}

func (m *resmgr) start(cfg interface{}) error {
	klog.Infof("starting resource manager...")
	if cfg != nil {
	}
	if err := m.nri.start(); err != nil {
		return err
	}
	return nil
}
