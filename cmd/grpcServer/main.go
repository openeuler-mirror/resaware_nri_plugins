package main

// #cgo CFLAGS: -I../../pkg/podAffinityServer
// #cgo LDFLAGS: -L../../pkg/podAffinityServer -lnumafast -lnuma
// #include "numafast.h"
import "C"
import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	pb "numaadj.huawei.com/pkg/podAffinityServer/proto"
	"os/exec"
	"strconv"
	"strings"
	"unsafe"

	"google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/util/yaml"
)

var (
	numaNums int
	fa       [1024]int //并查集
)

type server struct {
	pb.UnimplementedPodAffinityAwareServer
}

type NumaInfo C.struct_numa_info
type NumaNode C.struct_numa_node

type PodInfo struct {
	Uid  string
	tids []int
	gids []int
}

func convertCpuSet(cpus []int) string {
	strArray := make([]string, len(cpus))

	for i, cpu := range cpus {
		strArray[i] = strconv.Itoa(cpu)
	}

	return strings.Join(strArray, ",")
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
	fmt.Printf("get Numa Node Info, numaNums is: %d", numaNums)

	return &resp, nil
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

func calculateGid(podinfos *[]PodInfo) {
	for idx := 0; idx < len(*podinfos); idx++ {
		//for _, podinfo := range *podinfos {
		podinfo := &(*podinfos)[idx]
		podinfo.gids = make([]int, 0)
		for _, tid := range podinfo.tids {
			gid := C.get_gid(C.int(tid))
			podinfo.gids = append(podinfo.gids, int(gid))
		}
	}
}

func hasPodAffinity(p1 *PodInfo, p2 *PodInfo) bool {
	for _, g1 := range p1.gids {
		if g1 == -1 {
			continue
		}
		for _, g2 := range p2.gids {
			if g1 == g2 {
				return true
			}
		}
	}

	for _, g1 := range p1.gids {
		if g1 == -1 {
			continue
		}
		for _, t2 := range p2.tids {
			if g1 == t2 {
				return true
			}
		}
	}

	for _, g2 := range p2.gids {
		if g2 == -1 {
			continue
		}
		for _, t1 := range p1.tids {
			if g2 == t1 {
				return true
			}
		}
	}

	return false
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

func (s *server) GetPodAffinity(ctx context.Context, affinityRequest *pb.AffinityRequest) (*pb.AffinityResponse, error) {
	pods := affinityRequest.Pods

	podinfos := getTidListForPods(pods) // 计算每个pod对应的线程id

	calculateGid(podinfos) // 计算每个pod的每个线程的亲缘关系
	fmt.Println("----------------get GetPodAffinity ------------------")
	for _, pod := range pods {
		fmt.Println(pod.PodName)
	}
	for _, podinfo := range *podinfos {
		fmt.Printf("pod Uid: %s \n", podinfo.Uid)
		fmt.Println("tids:")
		for _, tid := range podinfo.tids {
			fmt.Printf("%d ", tid)
		}
		fmt.Println()
		fmt.Println("gids:")
		for _, gid := range podinfo.gids {
			fmt.Printf("%d ", gid)
		}
		fmt.Println()
	}
	fmt.Println("-------------------------------------------------------")

	init_unionFindSet(len(*podinfos)) // 初始化并查集用于亲缘关系分组

	for i := 0; i < len(*podinfos); i++ { // 使用并查集进行分组
		for j := i + 1; j < len(*podinfos); j++ {
			if hasPodAffinity(&((*podinfos)[i]), &((*podinfos)[j])) {
				join(i, j)
			}
		}
	}

	// groups用于保存具有亲缘关系的pod分组
	groups := make(map[int][]int)
	for i := 0; i < len(*podinfos); i++ {
		g := find(i)
		if groups[g] == nil {
			groups[g] = make([]int, 0)
		}
		groups[g] = append(groups[g], i)
	}

	pp := make(map[string]*pb.Pod)
	for _, pod := range pods {
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

func sameGids(gids1 *[]int, gids2 *[]int) int {
	sum := 0
	for _, g1 := range *gids1 {
		for _, g2 := range *gids2 {
			if g1 == g2 {
				sum += 1
			}
		}
	}
	return sum
}

func main() {
	var containerId string
	flag.StringVar(&containerId, "id", "", "containerId,")
	flag.Parse()

	fmt.Println("run numafast server")

	listen, err := net.Listen("tcp", ":9090")

	if err != nil {
		log.Fatalf("create listen: %v", err)
	}

	grpcServer := grpc.NewServer()

	pb.RegisterPodAffinityAwareServer(grpcServer, &server{})

	err = grpcServer.Serve(listen)

	if err != nil {
		log.Fatalf("server: %v", err)
	}

}
