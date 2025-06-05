#include "oeaware/data/network_interface_data.h"
#include "oeaware/data_list.h"
#include "oeaware/oe_client.h"
#include "oeaware/utils.h"
#include <stdio.h>
#include "netrela.h"
#include <numa.h>
#include <stdlib.h>
#include <time.h>

static CTopic topic1 = { OE_NET_INTF_INFO,  OE_LOCAL_NET_AFFINITY, "process_affinity" };

static net_rela rels[1024];
static int rels_nums;


static void phraseData(const DataList *dataList) {
    ProcessNetAffinityDataList *data = static_cast<ProcessNetAffinityDataList *> (dataList ->data[0]);
    rels_nums = data->count;
    for (int n=0;n<data->count;n++) {
        net_rela ra;
        ra.pid_a = data->affinity[n].pid1;
        ra.pid_b = data->affinity[n].pid2;
        ra.level = data->affinity[n].level;
        rels[n] = ra;
    }
}

static void mockData(const DataList *dataList) {
    int dataCnt = rand() % 10;
    rels_nums = dataCnt;
    for (int n=0;n<dataCnt;n++) {
        net_rela ra;
        ra.pid_a = rand() % 10000;
        ra.pid_b = rand() % 10000;
        ra.level = rand() % 10000;
        rels[n] = ra;
    }
}


//void readProcessNetRela() {
//    if (rels_nums <= 0) {
//        return;
//    }
//    for (int i=0;i<rels_nums;i++){
//
//    }
//}


static int QueryNetLocalAffiInfo(const DataList *dataList) {
    if (dataList && dataList->len && dataList->data) {
            phraseData(dataList);
        }
    return 0;
}

int oeawareStart() {
    int ret = OeInit();
    if (ret != 0) {
        printf( " SDK(Analysis) Init failed , result: %d \n" , ret);
        return -1;
    }
    OeSubscribe(&topic1, QueryNetLocalAffiInfo);
    return 0;
}

void oeawareShutdown() {
    OeUnsubscribe(&topic1);
    OeClose();
}

net_rela* getDataPtr() {
    return rels;
}

int getDataNums() {
    return rels_nums;
}



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