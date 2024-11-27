## 🎡 <font color="blue">介绍</font>

本产品是一款基于Pod亲缘关系来动态调整节点资源的NRI插件。由于多处理器的服务器使用Numa架构，不同的Numa节点资源分配对Pod的性能是一个重要的影响因素。实验表明，Pod之间存在很多的网络互相行为，可以将他们视作一组具有亲缘关系的Pod。如果将这一组具有亲缘关系的Pod分配在统一Numa节点上，可以有效的降低他们的跨Numa访问次数，从而提升系统的吞吐率。



## ⛄️ <font color="blue">能力介绍</font>

识别Pod之间的亲缘关系，动态调整Pod的资源分配，使节点的资源分配更加合理，从而提高系统吞吐率。



## ⚙️ <font color="blue">基本概念</font>

+ **Pod**： kubernets中可以创建和管理的最小单元，也是NRI插件重新分配资源的目标。

+ **container**：容器，镜像运行的实例，Pod中可以包含多个容器，一个容器对应一个进程。容器是用来感知Pod之间亲缘关系的重要参数来源。

+ **亲缘关系**：指容器或Pod之间在一定程度上具有网络互访或内存互访的行为。

+ **net_rship**: 一个二进制程序，采用eBPF技术用来获取网络亲缘关系。

+ **crictl**：是kubernetes中的一个容器接口工具，可以用于分析容器对应的进程Id，在NRI插件中需要借助该命令行工具来解析出容器对应的进程Id。

  

## ✏️ <font color="blue">实现原理</font>

![img](https://img2023.cnblogs.com/blog/1464124/202302/1464124-20230207232805753-18780686.png)

NRI(Node Resource Interface)，是节点资源接口，支持将特定逻辑插入兼容OCI的运行时，从而在容器生命周期时间点执行OCI规定范围之外的操作。NRI插件有两种运行方式，一种是NRI插件作为一个独立的进程，通过NRI Socket与CRI运行时通信；第二种是把NRI插件部署成为一个Daemonset，更方便在kubernetes上管理。因此，这里的主要原理是通过分析Pod之间的亲缘关系，然后通过NRI插件调整节点资源，将具有亲缘关系的Pod调整到合适的Numa节点上。

## 🌲 <font color="blue">环境准备</font>

#### 环境要求

+ 安装好kubernetes
+ containerd version 大于 1.7
+ 操作系统：openeuler 22.03-SP4，内核版本：kernel-5.10.0-230.0.0.129.oe2203sp4.aarch64
+ 安装好crictl工具

#### 环境搭建

操作系统：启动参数增加：kpti=off men_sampling_on numa_icon=enable



## 🧾 <font color="blue">任务场景</font>

#### 任务场景

工作节点上至少需要两个Pod，如果这两个Pod具有网络亲缘关系，NRI插件将他们调整到一个numa节点上。测试用例提供了2个Pod，分别为wrk客户端和Nginx服务器，客户端给服务器发送网络请求，并输出压测结果。可以在wrk的deployment文件中传入wrk压测参数，并通过查看Pod日志来查看吞吐率。

#### 开发流程

1. 在openeluer 22.03-SP4上安装Go和C的开发环境。
2. 定义CRD，用于描述节点资源和分配。
3. 编写NRI插件和Operator用于调整Pod的节点资源。
4. 编写grpc服务器，用于感知Pod之间的网络亲缘关系，并和NRI插件中的grpc客户端交互，发送亲缘关系拓扑结构。



## 🚀 <font color="blue">调测验证</font>

##### 启用NRI插件之前

0. 克隆代码仓到本地

1. 部署测试用例：

   kubectl apply -f nginx-deployment.yaml

   kubectl apply -f wrk-deployment.yaml

   kubectl apply -f nginx-svc.yaml

2. 使用kubectl 命令查看wrk对应Pod的日志，日志的输出为压测的值

##### 启用NRI插件之后

1. 编译构建grpc服务器：make build_grpc_server （可选，代码仓已提供编译后的二进制文件）
2. 启动ebpf程序 ./net_rship -l -I 6000 -i 3000
3. 安装NRI插件：make plugin_install
4. 再次使用kubectl log命令查看wrk对应Pod日志，查看输出压测的值