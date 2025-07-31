package agent

import (
	"context"
	"flag"
	"fmt"
	"github.com/containerd/nri/pkg/api"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	"numaadj.huawei.com/pkg/agent/watch"
	"numaadj.huawei.com/pkg/apis/resaware/v1alpha1"
	"numaadj.huawei.com/pkg/policy"
	nf "numaadj.huawei.com/pkg/policy/numafast"
	"numaadj.huawei.com/pkg/prometheus"
	"numaadj.huawei.com/pkg/resctrl"
	"numaadj.huawei.com/pkg/typedef"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Option func(*Agent) error

var (
	defaultKubeconfig     string
	defaultConfigFile     string
	defaultCrdName        string
	defaultNamespace      string
	defaultGrpcIp         string
	defaultGrpcPort       string
	defaultReconcilerTime int
	defaultPolicy         int
)

const (
	NET_AFFINITY = iota
	LOAD_BALANCE
	GROUP_LABEL = "rcgroup"
)

func init() {
	flag.StringVar(&defaultKubeconfig, "kubeconfig", "", "kubeconfig file path, if emptyt then use in-cluster configuration")
	flag.StringVar(&defaultConfigFile, "config-file", "", "config file, used for monitor insetd of a CustomResources")
	flag.StringVar(&defaultNamespace, "config-namespace", "kube-system", "namespace for configuration CustomResources")
	flag.StringVar(&defaultCrdName, "config-crdname", "podafi", "name for configuration CustomerResouces")
	flag.StringVar(&defaultGrpcIp, "grpc-ip", "127.0.0.1", "the ip address of grpc server for numafast")
	flag.StringVar(&defaultGrpcPort, "grpc-port", "9090", "the port of grpc server for numafast")
	flag.IntVar(&defaultReconcilerTime, "reconcile-time", 300, "the reconcile time of numafast(seconds)")
	flag.IntVar(&defaultPolicy, "policy", NET_AFFINITY, "the default reconcile policy")
}

// ConfigInterface is used bu the agent to access config custom resources.
type ConfigInterface interface {
	SetKubeClient(cfg *rest.Config) error
	CreateWatch(ctx context.Context, ns, name string, gvr schema.GroupVersionResource) (watch.Interface, error)
	GetConfigCrd(ctx context.Context, ns, name string, gvr schema.GroupVersionResource) (*unstructured.Unstructured, error)
	UpdateConfigCrd(ctx context.Context, ns string, un *unstructured.Unstructured) (*unstructured.Unstructured, error)
	CreateConfigCrd(ctx context.Context, name, ns string) error
}

func CrdConfigInterface() ConfigInterface {
	return &configIf{
		kind: 1,
	}
}

func New(cfgIf ConfigInterface, options ...Option) (*Agent, error) {
	a := &Agent{
		nodeName:   os.Getenv("NODE_NAME"),
		kubeconfig: defaultKubeconfig,
		configFile: defaultConfigFile,
		namespace:  defaultNamespace,
		crdname:    defaultCrdName,
		cfgIf:      cfgIf,
		stopC:      make(chan struct{}),
	}

	for _, o := range options {
		if err := o(a); err != nil {
			return nil, fmt.Errorf("failed to create agent: %w", err)
		}
	}

	if a.nodeName == "" && a.configFile == "" {
		return nil, fmt.Errorf("failed to create agent: neither node name nor config file set")
	}

	return a, nil
}

func (a *Agent) Start(notifyFn NotifyFn) error {

	a.notifyFn = notifyFn

	err := a.setupClients()
	if err != nil {
		return err
	}

	err = a.setupCrdWatch()
	if err != nil {
		return err
	}

	err = a.setupPodWatch()
	if err != nil {
		return err
	}

	err = a.setupWhitelistWatch()
	if err != nil {
		return err
	}

	err = a.setupGrpc()
	if err != nil {
		return err
	}

	err = a.setupConfigCrd()
	if err != nil {
		klog.Warningf("faile to create config crd: %v\n", err)
	} else {
		klog.Info("create config crd success")
	}

	eventChanOf := func(w watch.Interface) <-chan watch.Event {
		if w == nil {
			return nil
		}
		return w.ResultChan()
	}

	stopCh := make(chan struct{})
	defer close(stopCh)

	err = a.initPodCache(err)
	if err != nil {
		return err
	}

	getWatcher(a.k8sCli).watchNode()
	getWatcher(a.k8sCli).watchNodeConfigMap()
	getWatcher(a.k8sCli).watchGroupConfigMap()

	go wait.Until(resctrl.SyncResCtrlGroupTasks, time.Duration(ReSyncTimeSecond)*time.Second, stopCh)

	go prometheus.StartMetricsServer()

	klog.Info("initialization configuration complete, starting up now...")

	//TODO: 这里定时器指定了5分钟调用一次numafast, 怎么改成动态配置定时器的时间，或者配置定时策略？
	tick := time.Tick(time.Second * time.Duration(defaultReconcilerTime))

	loadBalanceTick := time.Tick(time.Second * time.Duration(defaultReconcilerTime))

	for {
		select {
		case <-a.stopC:
			a.cleanUpWatches()
			return nil

		case <-tick:
			if defaultPolicy != NET_AFFINITY {
				break
			}
			if err := a.updateByNumafastAware(); err != nil {
				klog.Warningf("failed to update by numafaster Aware: %v", err)
			}

		case <-loadBalanceTick:
			if defaultPolicy != LOAD_BALANCE {
				break
			}
			if err := a.updateByLoadBalancing(); err != nil {
				klog.Warningf("failed to update by laod balancing: %v", err)
			}

		case e, ok := <-eventChanOf(a.crdWatch):
			klog.Info("cra watch ^-^")
			if !ok {
				klog.Warningf("can't accept event to handle crd config update, error: %v", e.Object)
				break
			}
			if e.Type == watch.Added || e.Type == watch.Modified {
				if err := a.updateByCrdConfig(e.Object); err != nil {
					klog.Warningf("failed to update by crd config: %v", err)
				}
			}

		case e, ok := <-eventChanOf(a.podWath):
			if !ok {
				klog.Warningf("can't accept event to handle event of pod watch , error: %v", e.Object)
			}
			if e.Type == watch.Deleted {
				if err := a.deleteObsoletePodsInCr(e); err != nil {
					klog.Warningf("failed to delete pod: %v", err)
				}
			}

		case e, ok := <-eventChanOf(a.whitelistWatch):
			klog.Info("waitlist watch ^-^")
			if !ok {
				klog.Warningf("can't accept event to handle whitelist update, error: %v", e.Object)
				break
			}
			if e.Type == watch.Added || e.Type == watch.Modified {
				if _, err := a.getManagedPods(); err != nil {
					klog.Warningf("failed to update by crd config: %v", err)
				}
			}
		}
	}
}

func (a *Agent) initPodCache(err error) error {
	pods, err := a.k8sCli.CoreV1().Pods("").List(context.TODO(), v1.ListOptions{
		LabelSelector: GROUP_LABEL,
	})
	if err != nil {
		klog.Errorf("failed to list pods: %v", err)
		return err
	}
	podCacheInstance := typedef.PodCacheInstance()
	for _, pod := range pods.Items {
		podCacheInstance.UpdatePod(&typedef.PodInfo{
			Name:      pod.Name,
			UID:       string(pod.UID),
			Namespace: pod.Namespace,
			Labels:    pod.Labels,
		})
	}
	return nil
}

func (a *Agent) Stop() {
	a.stopLock.Lock()
	defer a.stopLock.Unlock()

	if a.stopC != nil {
		close(a.stopC)
		_ = <-a.doneC
		a.stopC = nil
	}
}

func (a *Agent) getManagedPods() (*v1alpha1.Whitelist, error) {
	gvr := schema.GroupVersionResource{
		Group:    v1alpha1.SchemeGroupVersion.Group,
		Version:  "v1alpha1",
		Resource: "whitelists",
	}

	un, err := a.cfgIf.GetConfigCrd(context.Background(), a.namespace, "whitelist", gvr)
	if err != nil {
		return nil, err
	}

	whitelist := &v1alpha1.Whitelist{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(un.UnstructuredContent(), whitelist)
	if err != nil {
		return nil, err
	}

	return whitelist, nil
}

func (a *Agent) updateByLoadBalancing() error {
	numaInfo, err := a.grpcClient.GetNumaNodes()
	if err != nil {
		return err
	}

	managedPods, err := a.getManagedPods()

	whitelistPodMap := make(map[string]bool)
	for _, targetPods := range managedPods.Spec.TargetPods {
		for _, podName := range targetPods.Names {
			key := fmt.Sprintf("%s-%s", targetPods.Namespace, podName)
			whitelistPodMap[key] = true
		}
	}

	gvr := schema.GroupVersionResource{
		Group:    v1alpha1.SchemeGroupVersion.Group,
		Version:  "v1alpha1",
		Resource: "oenumas",
	}
	un, err := a.cfgIf.GetConfigCrd(context.Background(), a.namespace, a.crdname, gvr)
	if err != nil {
		return err
	}

	oenuma := &v1alpha1.Oenuma{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(un.UnstructuredContent(), oenuma)
	if err != nil {
		return err
	}

	var targetNode *v1alpha1.Node = nil
	for idx := 0; idx < len(oenuma.Spec.Node); idx++ {
		if a.nodeName == oenuma.Spec.Node[idx].Name { // 只能维护插件所在的工作节点上的Pod
			targetNode = &oenuma.Spec.Node[idx]
			break
		}
	}

	if targetNode == nil {
		targetNode = &v1alpha1.Node{
			Name:        a.nodeName,
			PodAffinity: make([]v1alpha1.PodAffinity, 0),
			Numa:        make([]v1alpha1.Numa, 0),
		}
		for _, numa := range numaInfo.NumaNodes {
			targetNode.Numa = append(targetNode.Numa, v1alpha1.Numa{
				NumaNum: int32(numa.NumaNumer),
				Cpuset:  numa.Cpuset,
				Memset:  numa.Memset,
			})
		}

		if oenuma.Spec.Node == nil {
			oenuma.Spec.Node = make([]v1alpha1.Node, 0)
		}
		oenuma.Spec.Node = append(oenuma.Spec.Node, *targetNode)
	}

	podList, err := a.k8sCli.CoreV1().Pods("").List(context.Background(), v1.ListOptions{
		FieldSelector: fmt.Sprintf("spec.nodeName=%s", a.nodeName),
	})

	targetNode.PodAffinity = make([]v1alpha1.PodAffinity, 0)
	numas := len(numaInfo.NumaNodes)
	nonGuaranteedPodNumaIdx := 0

	allCpus := make([]string, 0)
	for _, numaNode := range numaInfo.NumaNodes {
		allCpus = append(allCpus, strings.Split(numaNode.Cpuset, ",")...)
	}

	cpuNumberPerNuma := len(allCpus) / len(numaInfo.NumaNodes)
	guranteedPodCpuIdx := 0

	for _, pod := range podList.Items {
		key := fmt.Sprintf("%s-%s", pod.Namespace, pod.Name)
		if !whitelistPodMap[key] {
			continue
		}
		klog.Infof("%s, %s, %s", pod.Namespace, pod.Name, pod.Status.QOSClass)

		// 防止Pod中的容器的CPU出现跨numa节点
		overFlowCpuIdx := guranteedPodCpuIdx / cpuNumberPerNuma

		if pod.Status.QOSClass == "Guaranteed" { // Guaranteed的Pod需要按容器定义的cpu分配资源，其余的按numa分配资源
			numaNo := guranteedPodCpuIdx / cpuNumberPerNuma
			pa := v1alpha1.PodAffinity{
				NumaNum:    int32(numaNo),
				PodName:    pod.Name,
				Namespace:  pod.Namespace,
				Containers: make([]v1alpha1.Container, 0),
			}
			for _, container := range pod.Spec.Containers {
				cpuQuantity, _ := container.Resources.Requests.Cpu().AsInt64()
				containerId := ""
				for _, c := range pod.Status.ContainerStatuses {
					if c.Name == container.Name {
						containerId = c.ContainerID
						break
					}
				}

				cpus := make([]string, 0)
				for i := 0; int64(i) < cpuQuantity; i++ {
					//增量分配cpu，如果超出numa节点范围则在节点内分配
					if guranteedPodCpuIdx/cpuNumberPerNuma == numaNo {
						cpus = append(cpus, allCpus[guranteedPodCpuIdx])
						guranteedPodCpuIdx++
					} else {
						cpus = append(cpus, allCpus[overFlowCpuIdx])
						overFlowCpuIdx++
						if overFlowCpuIdx == (numaNo+1)*cpuNumberPerNuma {
							overFlowCpuIdx = numaNo * cpuNumberPerNuma
						}
					}
				}

				pa.Containers = append(pa.Containers, v1alpha1.Container{
					Name:        container.Name,
					ContainerId: containerId,
					Cpuset:      strings.Join(cpus, ","),
					Memset:      strconv.Itoa(numaNo),
				})
			}
			targetNode.PodAffinity = append(targetNode.PodAffinity, pa)
		} else {
			targetNode.PodAffinity = append(targetNode.PodAffinity, v1alpha1.PodAffinity{
				NumaNum:    int32(nonGuaranteedPodNumaIdx),
				PodName:    pod.Name,
				Namespace:  pod.Namespace,
				Containers: nil, //Containers没有分配具体资源表示占用整个numa
			})
			nonGuaranteedPodNumaIdx = (nonGuaranteedPodNumaIdx + 1) % numas
		}
	}

	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(oenuma)
	if err != nil {
		return err
	}
	un.Object = obj
	_, err = a.cfgIf.UpdateConfigCrd(context.Background(), a.namespace, un)
	if err != nil {
		return err
	}

	return nil
}

func (a *Agent) updateByNumafastAware() error {
	klog.Info("update pod node resource by numafast aware ...")

	gvr := schema.GroupVersionResource{
		Group:    v1alpha1.SchemeGroupVersion.Group,
		Version:  "v1alpha1",
		Resource: "oenumas",
	}
	un, err := a.cfgIf.GetConfigCrd(context.Background(), a.namespace, a.crdname, gvr)
	if err != nil {
		return err
	}

	oenuma := &v1alpha1.Oenuma{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(un.UnstructuredContent(), oenuma)
	if err != nil {
		return err
	}

	var targetNode *v1alpha1.Node = nil
	for idx := 0; idx < len(oenuma.Spec.Node); idx++ {
		if a.nodeName == oenuma.Spec.Node[idx].Name { // 只能维护插件所在的工作节点上的Pod
			targetNode = &oenuma.Spec.Node[idx]
			break
		}
	}

	numaInfo, err := a.grpcClient.GetNumaNodes()
	if err != nil {
		return err
	}
	// 如果工作节点上的Pod首次调整，新建一个Node的数据类型为其维护更新的内容和更新后的状态
	if targetNode == nil {
		return fmt.Errorf("faile to get targetNode")
	}

	podList, err := a.k8sCli.CoreV1().Pods("default").List(context.Background(), v1.ListOptions{})
	if err != nil {
		return fmt.Errorf("get pods list error: %v", err)
	}

	mps := make(map[string]string)
	for _, pod := range podList.Items {
		mps[pod.Name] = string(pod.Status.QOSClass)
	}

	affiRelaship, err := a.grpcClient.GetPodAffinityRelationship(podList)
	if err != nil {
		return err
	}
	fmt.Println("-------------get pod affinity affiRelaship ----------------")
	for _, group := range affiRelaship.AffinityGroup {
		fmt.Printf("group.NumaNumer: %v\n", group.NumaNumer)
		for _, pod := range group.Pods {
			fmt.Printf("pod.PodName: %v\npod.namespace: %v\n", pod.PodName, pod.PodNamespace)
		}
	}
	fmt.Println("-----------------------------------------------------------")

	for _, group := range affiRelaship.AffinityGroup {
		for _, pod := range group.Pods {
			var podsAdjusted bool = false
			// 1. 检查当前的亲缘关系是否以存在该Pod，如果存在调整numa编号即可
			for idx := 0; idx < len(targetNode.PodAffinity); idx++ {
				if targetNode.PodAffinity[idx].PodName == pod.PodName {
					podsAdjusted = true
					targetNode.PodAffinity[idx].NumaNum = int32(group.NumaNumer)
					break
				}
			}
			if !podsAdjusted {
				targetNode.PodAffinity = append(targetNode.PodAffinity, v1alpha1.PodAffinity{
					NumaNum:   int32(group.NumaNumer),
					PodName:   pod.PodName,
					Namespace: pod.PodNamespace,
				})
			}
		}
	}

	cpuAllocIdx := make(map[int]int, len(numaInfo.NumaNodes))

	for i := 0; i < len(targetNode.PodAffinity); i++ {
		podafi := &targetNode.PodAffinity[i]
		if mps[podafi.PodName] != "Guaranteed" { //非Guaranteed类型的Pod独占整个numa节点
			podafi.Containers = make([]v1alpha1.Container, 0)
			for j := 0; j < len(podList.Items); j++ {
				if podList.Items[j].Name != podafi.PodName || podList.Items[j].Namespace != podafi.Namespace {
					continue
				}
				p := &podList.Items[j]
				for _, c := range p.Spec.Containers {
					conainerId := ""
					for _, cc := range p.Status.ContainerStatuses {
						if c.Name == cc.Name {
							conainerId = cc.ContainerID
							break
						}
					}
					podafi.Containers = append(podafi.Containers, v1alpha1.Container{
						Memset:      targetNode.Numa[podafi.NumaNum].Memset,
						Cpuset:      targetNode.Numa[podafi.NumaNum].Cpuset,
						ContainerId: conainerId,
					})
				}
			}
			continue
		}
		//Guaranteed类型的Pod按申请的cpu数量进行分配Cpu
		cpuset := strings.Split(numaInfo.NumaNodes[podafi.NumaNum].Cpuset, ",")
		idx := cpuAllocIdx[int(podafi.NumaNum)]
		podafi.Containers = make([]v1alpha1.Container, 0)

		for j := 0; j < len(podList.Items); j++ {
			if podList.Items[j].Name != podafi.PodName || podList.Items[j].Namespace != podafi.Namespace {
				continue
			}
			p := &podList.Items[j]

			for _, c := range p.Spec.Containers {
				cpus := make([]string, 0)
				cpuQuantity, _ := c.Resources.Requests.Cpu().AsInt64()
				for k := 0; int64(k) < cpuQuantity; k++ {
					cpus = append(cpus, cpuset[idx])
					idx = (idx + 1) % len(cpuset)
				}
				containerId := ""
				for _, cc := range p.Status.ContainerStatuses {
					if c.Name == cc.Name {
						containerId = cc.ContainerID
						break
					}
				}

				podafi.Containers = append(podafi.Containers, v1alpha1.Container{
					Memset:      strconv.Itoa(int(podafi.NumaNum)),
					Cpuset:      strings.Join(cpus, ","),
					ContainerId: containerId,
					Name:        c.Name,
				})
			}
		}

		cpuAllocIdx[int(podafi.NumaNum)] = idx
	}

	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(oenuma)
	if err != nil {
		return err
	}
	un.Object = obj
	_, err = a.cfgIf.UpdateConfigCrd(context.Background(), a.namespace, un)
	if err != nil {
		return err
	}

	return nil
}

func (a *Agent) updateByCrdConfig(obj runtime.Object) error {
	if obj != nil {
		uobj, ok := obj.(*unstructured.Unstructured)
		if !ok {
			return fmt.Errorf("can not handle object %T, ignoring it.", obj)
		}

		oenuma := &v1alpha1.Oenuma{}
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(uobj.UnstructuredContent(), oenuma)
		if err != nil {
			return fmt.Errorf("failed to convert unstructured obj.")
		}

		if strings.ToUpper(oenuma.Spec.UpdateEnable) != strings.ToUpper("enable") {
			klog.Info("resource in crd oenuma can not be update")
			return nil
		}

		config, err := a.createCrdConfigPolicy(oenuma)
		if err != nil {
			return fmt.Errorf("failer to create crd config policy.")
		}
		return a.notifyFn(config)
	}
	return nil
}

func (a *Agent) createCrdConfigPolicy(oenuma *v1alpha1.Oenuma) (*policy.Config, error) {
	config := &policy.Config{}
	for _, node := range oenuma.Spec.Node {
		if node.Name != a.nodeName {
			continue //只更新本工作节点的pod
		}

		for _, podAfi := range node.PodAffinity {
			_, err := a.k8sCli.CoreV1().Pods(podAfi.Namespace).Get(context.TODO(), podAfi.PodName, v1.GetOptions{})
			if err != nil {
				klog.Errorf("can not found pod: %s in namespace: %s", podAfi.PodName, podAfi.Namespace)
				continue
			}

			for _, container := range podAfi.Containers {
				sp := strings.Split(container.ContainerId, "/")
				containerId := sp[len(sp)-1]

				containerUpdate := &api.ContainerUpdate{
					ContainerId: containerId,
					Linux:       &api.LinuxContainerUpdate{Resources: &api.LinuxResources{Cpu: &api.LinuxCPU{Cpus: "", Mems: ""}}},
				}
				containerUpdate.Linux.Resources.Cpu.Cpus = container.Cpuset
				containerUpdate.Linux.Resources.Cpu.Mems = container.Memset
				config.Push(containerUpdate)
			}
		}
	}
	return config, nil
}

type NotifyFn func(cfg interface{}) error

type Agent struct {
	nodeName   string
	kubeconfig string
	configFile string
	namespace  string
	crdname    string

	cfgIf          ConfigInterface // custom resource access interface
	k8sCli         *kubernetes.Clientset
	notifyFn       NotifyFn // config resource change notification callback
	crdWatch       watch.Interface
	whitelistWatch watch.Interface
	podWath        watch.Interface
	grpcClient     *nf.GrpcClient

	stopLock sync.Mutex
	stopC    chan struct{}
	doneC    chan struct{}
}

func (a *Agent) setupClients() error {
	cfg, err := a.getK8sConfig()
	if err != nil {
		return err
	}

	a.k8sCli, err = kubernetes.NewForConfig(cfg)
	if err != nil {
		return fmt.Errorf("failed to setup kubernetes client: %v", err)
	}

	err = a.cfgIf.SetKubeClient(cfg)
	if err != nil {
		return fmt.Errorf("failed to setup config resource client: %v", err)
	}
	return nil
}

func (a *Agent) getK8sConfig() (*rest.Config, error) {
	var (
		cfg *rest.Config
		err error
	)

	if a.kubeconfig == "" {
		cfg, err = rest.InClusterConfig()
	} else {
		cfg, err = clientcmd.BuildConfigFromFlags("", a.kubeconfig)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get kuberneters config: %v", err)
	}

	return cfg, nil
}

func (a *Agent) setupCrdWatch() error {
	crdWatch, err := watch.Object(context.Background(), a.namespace, a.crdname,
		func(ctx context.Context, ns, name string) (watch.Interface, error) {
			gvr := schema.GroupVersionResource{
				Group:    v1alpha1.SchemeGroupVersion.Group,
				Version:  "v1alpha1",
				Resource: "oenumas",
			}
			return a.cfgIf.CreateWatch(ctx, ns, name, gvr)
		},
	)

	if err != nil {
		return fmt.Errorf("failed to setup crd watch: %v", err)
	}

	a.crdWatch = crdWatch

	return nil
}

func (a *Agent) setupWhitelistWatch() error {
	whitelistWatch, err := watch.Object(context.Background(), a.namespace, "whitelist",
		func(ctx context.Context, ns, name string) (watch.Interface, error) {
			gvr := schema.GroupVersionResource{
				Group:    v1alpha1.SchemeGroupVersion.Group,
				Version:  "v1alpha1",
				Resource: "whitelists",
			}
			return a.cfgIf.CreateWatch(ctx, ns, name, gvr)
		},
	)

	if err != nil {
		return fmt.Errorf("failed to setup whitelist watch: %v", err)
	}
	a.whitelistWatch = whitelistWatch

	return nil
}

func (a *Agent) setupPodWatch() error {
	podWath, err := watch.Object(context.Background(), "", "",
		func(ctx context.Context, ns, name string) (watch.Interface, error) {
			return a.k8sCli.CoreV1().Pods("").Watch(context.Background(), v1.ListOptions{})
		},
	)

	if err != nil {
		return fmt.Errorf("failed to setup pod watch: %v", err)
	}
	a.podWath = podWath
	return nil
}

func (a *Agent) cleanUpWatches() {
	if a.crdWatch != nil {
		a.crdWatch.Stop()
		a.crdWatch = nil
	}

	if a.whitelistWatch != nil {
		a.whitelistWatch.Stop()
		a.whitelistWatch = nil
	}
}

func (a *Agent) setupGrpc() error {
	grpcClient, err := nf.NewGrpcClient(defaultGrpcIp, defaultGrpcPort)
	if err != nil {
		return fmt.Errorf("can not set up grpc client")
	}
	a.grpcClient = grpcClient
	return nil
}

func (a *Agent) setupConfigCrd() error {
	// 创建基础crd
	err := a.cfgIf.CreateConfigCrd(context.Background(), a.crdname, a.namespace)
	if err != nil {
		return err
	}

	gvr := schema.GroupVersionResource{
		Group:    v1alpha1.SchemeGroupVersion.Group,
		Version:  "v1alpha1",
		Resource: "oenumas",
	}
	un, err := a.cfgIf.GetConfigCrd(context.Background(), a.namespace, a.crdname, gvr)
	if err != nil {
		return err
	}

	oenuma := &v1alpha1.Oenuma{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(un.UnstructuredContent(), oenuma)
	if err != nil {
		return err
	}

	var targetNode *v1alpha1.Node = nil
	for idx := 0; idx < len(oenuma.Spec.Node); idx++ {
		if a.nodeName == oenuma.Spec.Node[idx].Name { // 只能维护插件所在的工作节点上的Pod
			targetNode = &oenuma.Spec.Node[idx]
		}
	}

	numaInfo, err := a.grpcClient.GetNumaNodes()
	if err != nil {
		return err
	}

	getNumaInfo := func(tn string) {
		logrus.Infof("get numa node for node: %s", tn)
		for _, numa := range numaInfo.NumaNodes {
			targetNode.Numa = append(targetNode.Numa, v1alpha1.Numa{
				NumaNum: int32(numa.NumaNumer),
				Cpuset:  numa.Cpuset,
				Memset:  numa.Memset,
			})
		}
	}

	// 如果工作节点上的Pod首次调整，新建一个Node的数据类型为其维护更新的内容和更新后的状态
	if targetNode == nil {
		targetNode = &v1alpha1.Node{
			Name:        a.nodeName,
			PodAffinity: make([]v1alpha1.PodAffinity, 0),
			Numa:        make([]v1alpha1.Numa, 0),
		}
		//for _, numa := range numaInfo.NumaNodes {
		//	targetNode.Numa = append(targetNode.Numa, v1alpha1.Numa{
		//		NumaNum: int32(numa.NumaNumer),
		//		Cpuset:  numa.Cpuset,
		//		Memset:  numa.Memset,
		//	})
		//}
		getNumaInfo("nil node")
		if oenuma.Spec.Node == nil {
			oenuma.Spec.Node = make([]v1alpha1.Node, 0)
		}
		oenuma.Spec.Node = append(oenuma.Spec.Node, *targetNode)
	} else {
		targetNode.PodAffinity = make([]v1alpha1.PodAffinity, 0)
		targetNode.Numa = make([]v1alpha1.Numa, 0)
		getNumaInfo(targetNode.Name)
	}

	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(oenuma)
	if err != nil {
		return err
	}
	un.Object = obj
	_, err = a.cfgIf.UpdateConfigCrd(context.Background(), a.namespace, un)
	if err != nil {
		return err
	}
	return nil
}

func (a *Agent) deleteObsoletePodsInCr(event watch.Event) error {
	pod, ok := event.Object.(*corev1.Pod)
	if !ok {
		return fmt.Errorf("can not translate to pod")
	}

	gvr := schema.GroupVersionResource{
		Group:    v1alpha1.SchemeGroupVersion.Group,                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                    
		Version:  "v1alpha1",
		Resource: "oenumas",
	}
	un, err := a.cfgIf.GetConfigCrd(context.Background(), a.namespace, a.crdname, gvr)
	if err != nil {
		return err
	}

	oenuma := &v1alpha1.Oenuma{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(un.UnstructuredContent(), oenuma)
	if err != nil {
		return err
	}

	var targetNode *v1alpha1.Node = nil
	for idx := 0; idx < len(oenuma.Spec.Node); idx++ {
		if a.nodeName == oenuma.Spec.Node[idx].Name { // 只能维护插件所在的工作节点上的Pod
			targetNode = &oenuma.Spec.Node[idx]
			break
		}
	}

	if targetNode == nil {
		logrus.Infof("current node is not in oenuma, oenuma's node: %+v", oenuma.Spec.Node)
		return nil
	}

	// 删除目标Pod
	j := 0
	for _, podafi := range targetNode.PodAffinity {
		if podafi.PodName != pod.Name || podafi.Namespace != pod.Namespace {
			targetNode.PodAffinity[j] = podafi
			j++
		}
	}
	targetNode.PodAffinity = targetNode.PodAffinity[:j]

	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(oenuma)
	if err != nil {
		return err
	}
	un.Object = obj
	_, err = a.cfgIf.UpdateConfigCrd(context.Background(), a.namespace, un)
	if err != nil {
		return err
	}

	return nil
}
