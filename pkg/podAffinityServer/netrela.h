#ifndef __NETRELA_H_
#define __NETRELA_H_
#ifdef __cplusplus
extern "C" {
#endif

typedef struct numa_node {
  int cpu_quantity;
  int* cpuset;
  int memset;
} numa_node;


typedef struct numa_info {
  int node_quantity;
  numa_node* nodes;
} numa_info;

numa_info get_numa_info();

typedef struct net_rela {
    int pid_a;
    int pid_b;
    int level;
} net_rela;

int oeawareStart();
void oeawareShutDown();
net_rela* getDataPtr();
int getDataNums();

#ifdef __cplusplus
}
#endif

#endif