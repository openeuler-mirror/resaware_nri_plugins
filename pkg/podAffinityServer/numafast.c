#include "numafast.h"
#include <numa.h>
#include <stdlib.h>
#include <stdio.h>
#include <linux/sched_ctrl.h>
#include <unistd.h>
#include <sys/ioctl.h>
#include <sys/types.h>
#include <fcntl.h>

//static numa_info ni;


numa_info get_numa_info() {
  numa_info ni;
  
  int max_node = numa_max_node();
  int node, cpu;

  ni.node_quantity = max_node + 1;
  ni.nodes = (numa_node*)malloc(sizeof(numa_node) * ni.node_quantity);

  for (node = 0; node <= max_node;node++) {
    numa_node n;
    n.cpu_quantity = 0;
    n.cpuset = (int*)malloc(sizeof(int) * 64);
    n.memset = node;

    for (cpu=0;cpu<numa_num_configured_cpus();cpu++) {
      if(numa_node_of_cpu(cpu) == node) {
        n.cpuset[n.cpu_quantity] = cpu;
        ++n.cpu_quantity;
      }
    }
    ni.nodes[node] = n;
  }

  return ni;
}

//返回pid对应的gid，-1表示没有感知到网络亲缘关系
int get_gid(int tid) {
  int fd;
  if ((fd=open("/dev/relationship_ctrl",O_RDWR))==-1) {
    return -1;
  }
  struct sctl_get_relationship_args sgra;
  sgra.tid = tid;
  int ret;
  ret = ioctl(fd, SCTL_GET_RSHIP,&sgra);
  close(fd);
  if (ret == -1 || sgra.nrsi.valid == -1) {
    return -1;
  }

  struct sctl_net_relationship_info net_rela = sgra.nrsi;
  if (net_rela.grp_hdr.nr_tasks < 2) {
    return -1;
  }
  return net_rela.grp_hdr.gid;
}

