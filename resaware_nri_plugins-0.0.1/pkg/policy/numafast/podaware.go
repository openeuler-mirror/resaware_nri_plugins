package numafast

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	v1 "k8s.io/api/core/v1"
	pb "numaadj.huawei.com/pkg/policy/numafast/proto"
)

type GrpcClient struct {
	ip     string
	port   string
	conn   *grpc.ClientConn
	client pb.PodAffinityAwareClient
}

func NewGrpcClient(ip, port string) (*GrpcClient, error) {
	gc := GrpcClient{
		ip:   ip,
		port: port,
	}
	conn, err := grpc.NewClient(gc.ip+":"+gc.port, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("did not connect:%v", err)
	}
	gc.conn = conn

	client := pb.NewPodAffinityAwareClient(conn)
	gc.client = client
	return &gc, nil
}

func (c *GrpcClient) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

func (c *GrpcClient) GetNumaNodes() (*pb.NumaInfoResponse, error) {
	numaInfo, err := c.client.GetNumaNodeInof(context.Background(), nil)
	if err != nil {
		return nil, err
	}
	return numaInfo, nil
}

func (c *GrpcClient) GetPodAffinityRelationship(podList *v1.PodList) (*pb.AffinityResponse, error) {
	pods := []*pb.Pod{}

	for _, pod := range podList.Items {
		p := pb.Pod{
			Uid:           string(pod.UID),
			PodName:       pod.Name,
			PodNamespace:  pod.Namespace,
			ContainerdIDs: make([]string, 0),
		}
		for _, c := range pod.Status.ContainerStatuses {
			p.ContainerdIDs = append(p.ContainerdIDs, c.ContainerID)
		}
		pods = append(pods, &p)
	}

	request := pb.AffinityRequest{
		Pods: pods,
	}

	resp, err := c.client.GetPodAffinity(context.Background(), &request)
	if err != nil {
		return nil, fmt.Errorf("get pod affinity error: %v", err)
	}

	return resp, nil
}
