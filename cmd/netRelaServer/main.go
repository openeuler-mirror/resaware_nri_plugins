package main

// #cgo CFLAGS: -I../../pkg/podAffinityServer
// #cgo LDFLAGS: -L../../pkg/podAffinityServer -lnetrela -loeaware-sdk -lnuma
// #include "netrela.h"
import "C"
import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	"k8s.io/klog/v2"
	"net"
	pb "numaadj.huawei.com/pkg/podAffinityServer/proto"
	"os/exec"
	"strconv"
	"strings"
	"unsafe"

	"k8s.io/apimachinery/pkg/util/yaml"
)

var (
	numaNums int
	fa       [1024]int //并查集
)

type netRela C.struct_net_rela

type PodInfo struct {
	Uid  string
	tids []int
}

type server struct {
	pb.UnimplementedPodAffinityAwareServer
}

func main() {
	klog.Info("run net affinity server .... ")
	listen, err := net.Listen("tcp", ":9090")
	if err != nil {
		klog.Fatalf("Failed to create listener: %v", err)
	}

	s := &server{}
	if err := s.ConnectOeAware(); err != nil {
		klog.Fatalf("Failed to start server: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterPodAffinityAwareServer(grpcServer, s)

	err = grpcServer.Serve(listen)
	if err != nil {
		klog.Fatalf("Failed to start server: %v", err)
	}
}

func (s *server) ConnectOeAware() error {
	ret := int(C.oeawareStart())
	if ret != 0 {
		return fmt.Errorf("failed to subscribe oeaware, exit code: %d", ret)
	}
	return nil
}

func (s *server) GetPodAffinity(ctx context.Context, affinityRequest *pb.AffinityRequest) (*pb.AffinityResponse, error) {
	dataPtr := (*[1024]netRela)(unsafe.Pointer(C.getDataPtr()))[:1024:1024]
	n := int(C.getDataNums())
	klog.Infof("detect %d set of net affinity process\n", n)
	rls := make([]netRela, 0)
	for i := 0; i < n; i++ {
		var rl netRela
		rl.pid_a = dataPtr[i].pid_a
		rl.pid_b = dataPtr[i].pid_b
		rl.level = dataPtr[i].level
		rls = append(rls, rl)
		klog.V(4).Infof("Oeaware: => %d %d %d\n", rl.pid_a, rl.pid_b, rl.level)
	}

	pods := affinityRequest.Pods
	podinfos := getTidListForPods(pods)
	initNetRelationship(podinfos, &rls)

	return getPodsAffinity(affinityRequest, podinfos)
}

func getPodsAffinity(affinityRequest *pb.AffinityRequest, podinfos *[]PodInfo) (*pb.AffinityResponse, error) {
	groups := make(map[int][]int)
	for i := 0; i < len(*podinfos); i++ {
		g := find(i)
		if groups[g] == nil {
			groups[g] = make([]int, 0)
		}
		groups[g] = append(groups[g], i)
	}
	pp := make(map[string]*pb.Pod)

	for _, pod := range affinityRequest.Pods {
		pp[pod.Uid] = pod
	}

	resp := &pb.AffinityResponse{}
	numaNode := 0
	for _, idxs := range groups {
		if len(idxs) < 2 { // 没有和他有亲缘关系的pod
			continue
		}
		affiGroup := pb.AffinityGroup{}
		affiGroup.NumaNumer = int64(numaNode)
		numaNode = (numaNode + 1) % numaNums

		if affiGroup.Pods == nil {
			affiGroup.Pods = make([]*pb.Pod, 0)
		}

		for _, idx := range idxs {
			uid := (*podinfos)[idx].Uid
			affiGroup.Pods = append(affiGroup.Pods, &pb.Pod{
				Uid:           uid,
				PodName:       pp[uid].PodName,
				PodNamespace:  pp[uid].PodNamespace,
				ContainerdIDs: pp[uid].ContainerdIDs,
			})
		}

		if resp.AffinityGroup == nil {
			resp.AffinityGroup = make([]*pb.AffinityGroup, 0)
		}
		resp.AffinityGroup = append(resp.AffinityGroup, &affiGroup)
	}

	return resp, nil
}

func initNetRelationship(podinfos *[]PodInfo, rls *[]netRela) {
	gah := generateNetRelaGraph(rls)
	hasPodAffinity := func(p1 *PodInfo, p2 *PodInfo, gah *map[int]map[int]struct{}) bool {
		for _, t1 := range p1.tids {
			for _, t2 := range p2.tids {
				if _, exists := (*gah)[t1][t2]; exists {
					return true
				}
				if _, exists := (*gah)[t2][t1]; exists {
					return true
				}
			}
		}
		return false
	}
	init_unionFindSet(len(*podinfos))
	for i := 0; i < len(*podinfos); i++ { // 使用并查集进行分组
		for j := i + 1; j < len(*podinfos); j++ {
			if hasPodAffinity(&((*podinfos)[i]), &((*podinfos)[j]), gah) {
				join(i, j)
			}
		}
	}
}

func generateNetRelaGraph(rls *[]netRela) *map[int]map[int]struct{} {
	gah := make(map[int]map[int]struct{})
	for _, r := range *rls {
		pida := int(r.pid_a)
		pidb := int(r.pid_b)
		if (gah[pida]) == nil {
			gah[pida] = make(map[int]struct{})
		}
		gah[pida][pidb] = struct{}{}
		if (gah[pidb]) == nil {
			gah[pidb] = make(map[int]struct{})
		}
		gah[pida][pidb] = struct{}{}
	}
	return &gah
}

func getTidListForPods(pods []*pb.Pod) *[]PodInfo {
	podInfos := make([]PodInfo, 0)
	for _, pod := range pods {
		var pinfo PodInfo
		pinfo.Uid = pod.Uid
		pinfo.tids = make([]int, 0)
		for _, cid := range pod.ContainerdIDs {
			pid, err := getPidByContainerdId(cid)
			if err != nil {
				fmt.Printf("解析容器：%s对应的进程id错误，%v\n", cid, err)
			} else {
				pinfo.getTids(pid)
			}
		}
		podInfos = append(podInfos, pinfo)
	}

	return &podInfos
}

func (pi *PodInfo) getTids(targetPid int) {
	pi.tids = append(pi.tids, targetPid)
	cmd := exec.Command("sh", "-c", "ps -efT | grep "+strconv.Itoa(targetPid))
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("获取进程%d的子进程或线程失败:%v\n", targetPid, err)
		return
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		pid, err := strconv.Atoi(fields[2])
		if err != nil {
			continue
		}

		ppid, err := strconv.Atoi(fields[3])
		if err != nil {
			continue
		}

		if ppid == targetPid {
			pi.tids = append(pi.tids, pid)
		}
	}
}

type Config struct {
	Info struct {
		Pid int `yaml:"pid"`
	}
}

// 调用ctrl命令来获取container对应的进程Pid
func getPidByContainerdId(containerId string) (int, error) {
	parts := strings.SplitAfter(containerId, "://")
	if len(parts) > 1 {
		containerId = parts[1]
	} else {
		return -1, fmt.Errorf("containerId's fommat is error")
	}
	cmd := exec.Command("crictl", "inspect", "--output", "yaml", containerId)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return -1, err
	}
	var config Config
	err = yaml.Unmarshal(output, &config)
	if err != nil {
		return -1, err
	}
	return config.Info.Pid, nil
}

func init_unionFindSet(n int) {
	for i := 0; i <= n; i++ {
		fa[i] = i
	}
}

func find(x int) int {
	if x != fa[x] {
		fa[x] = find(fa[x])
	}
	return fa[x]
}

func join(x int, y int) {
	rx := find(x)
	ry := find(y)
	if rx != ry {
		fa[rx] = ry
	}
}

type NumaInfo C.struct_numa_info
type NumaNode C.struct_numa_node

func (s *server) GetNumaNodeInof(context.Context, *pb.NullRequest) (*pb.NumaInfoResponse, error) {
	ni := NumaInfo(C.get_numa_info())
	resp := pb.NumaInfoResponse{}
	resp.Valid = true
	resp.NumaNodes = make([]*pb.NumaNode, 0)

	pnodes := (*[1024]NumaNode)(unsafe.Pointer(ni.nodes))

	for i := 0; i < int(ni.node_quantity); i++ {
		node := pnodes[i]

		cpus := make([]int, 0)

		cs := (*[1024]C.int)(unsafe.Pointer(node.cpuset))
		for c := 0; c < int(node.cpu_quantity); c++ {
			cpus = append(cpus, int(cs[c]))
		}

		resp.NumaNodes = append(resp.NumaNodes, &pb.NumaNode{
			NumaNumer: int64(i),
			Memset:    strconv.Itoa(int(node.memset)),
			Cpuset:    convertCpuSet(cpus),
		})
	}

	numaNums = int(ni.node_quantity)
	fmt.Printf("get Numa Node Info, numaNums is: %d\n", numaNums)

	return &resp, nil
}

func convertCpuSet(cpus []int) string {
	strArray := make([]string, len(cpus))

	for i, cpu := range cpus {
		strArray[i] = strconv.Itoa(cpu)
	}

	return strings.Join(strArray, ",")
}
