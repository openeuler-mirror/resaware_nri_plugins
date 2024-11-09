package main

import (
	"flag"

	"k8s.io/klog/v2"
	"numaadj.huawei.com/pkg/agent"
	"numaadj.huawei.com/pkg/resmgr"
)

func main() {
	flag.Parse()
	agt, err := agent.New(agent.CrdConfigInterface())
	if err != nil {
		klog.Fatalf("%v", err)
	}

	mgr, err := resmgr.NewResourceManager(agt, nil)

	if err != nil {
		klog.Fatalf("%v", err)
	}

	if err := mgr.Start(); err != nil {
		klog.Fatalf("%v", err)
	}
}
