package resmgr

import (
	"flag"
	"time"

	nri "github.com/containerd/nri/pkg/api"
)

const (
	defaultPluginName  = "numaadj"
	defaultPluginIndex = "00"
)

type options struct {
	RebalaceTimer time.Duration
	NriPluginName string
	NriPluginIdx  string
	NriSocket     string
}

var opt = options{}

func init() {
	flag.StringVar(&opt.NriPluginName, "nri-plugin-name", defaultPluginName, "NRI plugin name to register.")
	flag.StringVar(&opt.NriPluginIdx, "nri-plugin-index", defaultPluginIndex, "NRI plugin index to register.")
	flag.StringVar(&opt.NriSocket, "nri-socket", nri.DefaultSocketPath, "NRI unix domain socker path to connect to.")
}
