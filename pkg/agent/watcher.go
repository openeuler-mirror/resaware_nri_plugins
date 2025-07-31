package agent

import (
	"context"
	"io/ioutil"
	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
	"numaadj.huawei.com/pkg/resctrl"
	"os"
	"path/filepath"
	"sigs.k8s.io/yaml"
	"strconv"
	"strings"
	"sync"
)

// the namespace of the ConfigMaps
const (
	defaultConfigMapNamespace = "rc-config"
	resCtrlSchemataFile       = "schemata"
	defaultQosConfigMapPrefix = "mpam-qos-config"
	defaultReSyncTimeSecond   = 10
)

var (
	resCtrlRoot         = os.Getenv("RESCTRL_PATH")
	configMapNamespace  = os.Getenv("CONFIGMAP_NAMESPACE")
	qosConfigMapPrefix  = os.Getenv("QOS_CONFIGMAP_NAME")
	ReSyncTimeSecond, _ = strconv.Atoi(os.Getenv("RE_SYNC_TIME"))
	nodeName            string
)

func init() {
	if configMapNamespace == "" {
		configMapNamespace = defaultConfigMapNamespace
	}
	if qosConfigMapPrefix == "" {
		qosConfigMapPrefix = defaultQosConfigMapPrefix
	}
	if ReSyncTimeSecond <= 0 || ReSyncTimeSecond > 3600 {
		ReSyncTimeSecond = defaultReSyncTimeSecond
	}
	if resCtrlRoot == "" {
		resCtrlRoot = "/sys/fs/resctrl"
	}
	nodeName = os.Getenv("NODE_NAME")

	if nodeName == "" {
		if hostname, err := ioutil.ReadFile("/etc/hostname"); err == nil {
			nodeName = strings.ToLower(strings.TrimSpace(string(hostname)))
		}
	}
}

// ConfigData resource control configuration
type ConfigData map[string]string

type watcher struct {
	k8sCli             kubernetes.Interface // k8s client interface
	ResCtrlConfigWatch watch.Interface
	sync.RWMutex
	nodeCfg   *ConfigData
	groupCfg  *ConfigData
	groupName string
}

var (
	singletonWatcher *watcher
	once             sync.Once
)

// getWatcher returns singleton k8s watcher instance.
func getWatcher(k8sCli kubernetes.Interface) *watcher {
	once.Do(func() {
		singletonWatcher = &watcher{
			k8sCli: k8sCli,
		}
	})
	return singletonWatcher
}

func (w *watcher) watchNode() {
	// watch this Node
	selector := meta.ListOptions{FieldSelector: "metadata.name=" + nodeName}
	k8w, err := w.k8sCli.CoreV1().Nodes().Watch(context.TODO(), selector)
	if err != nil {
		klog.Errorf("Failed to watch node (%q): %v", nodeName, err)
		return
	}

	go func(ev <-chan watch.Event, group string) {
		for e := range ev {
			switch e.Type {
			case watch.Error:
				klog.Warningf("Watch node error, event: %+v", e.Object)
				break
			case watch.Added, watch.Modified:
				klog.Infof("node (%s) is updated", nodeName)
				label, _ := e.Object.(*core.Node).Labels["ngroup"]

				// if the node group is changed, we start to watch the config of the new node group
				if group != label {
					group = label
					klog.Infof("node group is set to %s", group)
					w.Lock()
					w.groupName = group
					w.Unlock()
					if w.ResCtrlConfigWatch != nil {
						w.ResCtrlConfigWatch.Stop()
					}
				}
			case watch.Deleted:
				klog.Warning("our node is removed...")
			default:
				klog.Info("other event type")
			}
		}

		klog.Warning("seems node watcher is closed, going to restart ...")
		w.watchNode()
		klog.Warning("node configMap watcher restarted")
	}(k8w.ResultChan(), "")
}

func (w *watcher) watchNodeConfigMap() {
	// watch "rc-config.node.{NODE_NAME}" ConfigMap
	selector := meta.ListOptions{FieldSelector: "metadata.name=" + defaultQosConfigMapPrefix + ".node." + nodeName}
	k8w, err := w.k8sCli.CoreV1().ConfigMaps(configMapNamespace).Watch(context.TODO(), selector)
	if err != nil {
		klog.Errorf("Failed to watch ConfigMap rc-config.node.%q: %v", nodeName, err)
		return
	}

	go func(ev <-chan watch.Event) {
		for e := range ev {
			switch e.Type {
			case watch.Error:
				klog.Warningf("Watch node config error, event: %+v", e.Object)
				break
			case watch.Added, watch.Modified:
				klog.Info("ConfigMap rc-config.node." + nodeName + " is updated")
				cm, ok := e.Object.(*core.ConfigMap)
				if !ok {
					klog.Warning("It's not ok for type *core.ConfigMap")
					continue
				}
				w.setNodeConfig(&cm.Data)
			case watch.Deleted:
				klog.Info("ConfigMap rc-config.node." + nodeName + " is deleted")
				w.setNodeConfig(nil)
			default:
				klog.Info("other event type")
			}
		}

		klog.Warning("seems node configMap watcher is closed, going to restart ...")
		w.watchNodeConfigMap()
		klog.Warning("node configMap watcher restarted")
	}(k8w.ResultChan())
}

func (w *watcher) watchGroupConfigMap() {

	// watch group ConfigMap
	cmName := defaultQosConfigMapPrefix + ".default"
	w.Lock()
	if w.groupName != "" {
		cmName = defaultQosConfigMapPrefix + ".group." + w.groupName
	}
	w.Unlock()
	selector := meta.ListOptions{FieldSelector: "metadata.name=" + cmName}
	k8w, err := w.k8sCli.CoreV1().ConfigMaps(configMapNamespace).Watch(context.TODO(), selector)
	if err != nil {
		klog.Errorf("Failed to watch group ConfigMap (%q): %v", cmName, err)
		return
	}

	w.ResCtrlConfigWatch = k8w
	klog.Info("start watching ConfigMap " + cmName)

	go func(ev <-chan watch.Event) {
		for e := range ev {
			switch e.Type {
			case watch.Error:
				klog.Warningf("Watch group config error, event: %+v", e.Object)
				break
			case watch.Added, watch.Modified:
				cm, ok := e.Object.(*core.ConfigMap)
				if !ok {
					klog.Warning("It's not ok for type *core.ConfigMap")
					continue
				}
				klog.Infof("group ConfigMap (%s) is updated", cm.Name)
				w.setGroupConfig(&cm.Data)
			case watch.Deleted:
				cm, ok := e.Object.(*core.ConfigMap)
				if !ok {
					klog.Warning("It's not ok for type *core.ConfigMap")
					continue
				}
				klog.Infof("group ConfigMap (%s) is deleted, will try to apply default configmap", cm.Name)

				defaultCm, err := w.k8sCli.CoreV1().ConfigMaps(configMapNamespace).Get(context.TODO(), defaultQosConfigMapPrefix+".default", meta.GetOptions{})
				if err != nil {
					klog.Warningf("Failed to get default ConfigMap (%q): %v", cmName, err)
					w.setGroupConfig(nil)
					return
				}
				w.setGroupConfig(&defaultCm.Data)
			default:
				klog.Info("other event type")
			}
		}

		klog.Warning("seems group configMap watcher is closed, going to restart ...")
		w.watchGroupConfigMap()
		klog.Warning("group configMap watcher is restarted")

	}(k8w.ResultChan())
}

// set node-specific configuration
func (w *watcher) setNodeConfig(data *map[string]string) {
	w.Lock()
	defer w.Unlock()

	w.nodeCfg = (*ConfigData)(data)
	w.applyConfig()
}

// set group-specific or default configuration
func (w *watcher) setGroupConfig(data *map[string]string) {
	w.Lock()
	defer w.Unlock()

	w.groupCfg = (*ConfigData)(data)

	if w.nodeCfg == nil {
		w.applyConfig()
	}
}

func (w *watcher) applyConfig() {
	klog.Info("apply configuration")

	config := w.groupCfg

	if w.nodeCfg != nil {
		config = w.nodeCfg
	}

	if config == nil {
		klog.Warning("There is no configuration")
	}

	ApplyConfig(config)
}

type MpamConfig struct {
	MPAM map[string]GroupConfig `yaml:"mpam"`
}

type GroupConfig struct {
	L3PRI int `yaml:"L3PRI"`
	MBPRI int `yaml:"MBPRI"`
}

func ApplyConfig(data *ConfigData) {
	klog.Info("starting to apply configuration")
	var mpamGroups []string

	if data == nil {
		klog.Info("config data is nil")
		cleanResCtrlGroup(mpamGroups)
		return
	}

	var mpamConfig MpamConfig
	for _, val := range *data {
		klog.Infof("configdata value: \n%+v", val)
		err := yaml.Unmarshal([]byte(val), &mpamConfig)
		if err != nil {
			klog.Errorf("error parsing YAML: %+v", err)
			var yamlGroups []string
			cleanResCtrlGroup(yamlGroups)
		}
		klog.Infof("mpamGroups: %+v", mpamConfig)
		for groupName, group := range mpamConfig.MPAM {
			klog.Infof("group name: %s, group: %+v", groupName, group)
			if groupName == "DEFAULT" {
				groupName = ""
			}
			err := resctrl.NewMpamResCtrlGroup(groupName, group.L3PRI, group.MBPRI).SetGroupQos()
			if err != nil {
				klog.Errorf("failed to write resctrl schemata: %v", err)
				return
			}
		}
	}
}

// cleanResCtrlGroup removes resctrl group that not in 'groups'
func cleanResCtrlGroup(groups []string) {
	fis, err := ioutil.ReadDir(resCtrlRoot)
	if err != nil {
		klog.Errorf("resCtrlRoot is not exist, please ensure resctrl fs has been mounted")
		return
	}

	groupSet := make(map[string]bool)
	for _, group := range groups {
		groupSet[group] = true
	}

	for _, fi := range fis {
		if !fi.IsDir() {
			continue
		}

		if !groupSet[fi.Name()] {
			path := filepath.Join(resCtrlRoot, fi.Name())
			err := os.Remove(path)
			if err != nil {
				klog.Errorf("failed to remove %s: %v", path, err)
			} else {
				klog.Infof("%s is removed", path)
			}
		}
	}
}