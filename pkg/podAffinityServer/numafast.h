#ifndef  __NUMAFAST_H
#define __NUMAFAST_H

typedef struct numa_node {
  int cpu_quantity;
  int* cpuset;
  int memset;
} numa_node;


typedef struct numa_info {
  int node_quantity;
  numa_node* nodes;
} numa_info;

// typedef struct pod {
//   char* uid;
//   int* tids;
//   int* gids;
// }
// 
numa_info get_numa_info();

int get_gid(int tid);

#endif // ! __NUMAFAST_H
