#! /bin/bash
apply_config_crd() {
  local CRD_NAME="oenumas.resource.sop.huawei.com"
  if kubectl get crd | grep -wq "$CRD_NAME"; then
    echo "crd '$CRD_NAME' already exists."
  else
    # apply_crd
    kubectl apply -f /etc/nriplugin/resource.sop.huawei.com_oenumas.yaml
    if [ $? -eq 0 ]; then
      echo "crd '$CRD_NAME' create succecss."
    else
      echo "crd '$CRD_NAME' create failed."
    fi
  fi
}

apply_whitelist_crd() {
  local WHITE_LIST_NAME="whitelists.resource.sop.huawei.com"
  if kubectl get crd | grep -wq "$WHITE_LIST_NAME"; then
    echo "crd '$WHITE_LIST_NAME' already exists."
  else
    # apply_crd
    kubectl apply -f /etc/nriplugin/resource.sop.huawei.com_whitelists.yaml
    if [ $? -eq 0 ]; then
      echo "crd '$WHITE_LIST_NAME' create succecss."
    else
      echo "crd '$WHITE_LIST_NAME' create failed."
    fi
  fi
}

create_cr_namespace() {
  local CR_NAMESPACE="$1"

  if kubectl get namespace | grep -wq "$CR_NAMESPACE"; then
    echo "namespace '$CR_NAMESPACE' already exists."
  else
    # 创建命名空间
    kubectl create namespace "$CR_NAMESPACE"
    if [ $? -eq 0 ]; then
      echo "namespace '$CR_NAMESPACE' create succecss."
    else
      echo "namespace '$CR_NAMESPACE' create failed."
    fi
  fi
}

start_grpc_server() {
  local GRPC_SERVER_PATH="$1"
  if [ ! -f "$GRPC_SERVER_PATH" ]; then
    echo "grpc server not exists, begian to build grpc_server."
    make grpc_clean
    make build_grpc_server
    make grpc_clean
    if [ $? -ne 0 ]; then
      echo "can not build grpc server."
      exit 1
    else
      echo "build grpc server success."
    fi
  fi

  echo "start grpc server"
  "$GRPC_SERVER_PATH" &
  if [ $? -eq 0 ]; then
    echo "grpc server start success."
  else
    echo "grpc server start failed."
    exit 1
  fi
}

apply_nri_plugin() {
  local DAEMONSET_PATH="$1"
  if [ ! -f "$DAEMONSET_PATH" ]; then
    echo "can't not find nri plugin deploy file."
  else
    echo "$DAEMONSET_PATH"
    kubectl apply -f "$DAEMONSET_PATH"
  fi
}

plugin_install() {
  echo "install nri plugin."
  # 1. 创建crd
  echo "step 1: apply crd"
  apply_config_crd
  apply_whitelist_crd

  echo "step 2: create namespace"
  # 2. 创建cr namespace
  create_cr_namespace "tcs"

  # 3. 部署NRI插件
  # 3.1 启动grpc服务器
  echo "step3: start grpc server"
  start_grpc_server "./bin/grpc_server"

  # 3.2 安装NRI插件
  echo "step4: apply nri plugin"
  apply_nri_plugin "./deploy/numa-daemonset.yaml"

  if [ $? -eq 0 ]; then
    echo "nri plugin installed success."
  else
    echo "nri plugin installed failed."
  fi
}

plugin_uninstall() {
  echo "uninstall nri plugin."
  pids=$(pgrep -f grpc_server)

  #1. stop grpc server
  if [ -z "$pids" ]; then
    echo "grpc server not found, kill grpc server step skip...."
  else
    echo "kill the porcess of grpc server...."
    kill $pids

    if [ $? -eq 0 ]; then
      echo "grpc server stop success."
    else
      echo "grpc server stop failed."
    fi
  fi

  #2. uninstall nri plugin
  kubectl delete daemonset numaadj
  kubectl delete role oenuma-watch -n tcs
  kubectl delete rolebinding oenuma-watch-binding -n tcs
  kubectl delete clusterrole pod-watch -n default
  kubectl delete clusterrolebinding pod-watch-binding -n default
  kubectl delete oenuma podafi -n tcs
  kubectl delete namespace tcs
  kubectl delete crd oenumas.resource.sop.huawei.com
  kubectl delete crd whitelists.resource.sop.huawei.com
  exit 0
}

test_case_install() {
  echo "install test case."
  kubectl apply -f ./test/wrk/nginx-deployment.yaml
  kubectl apply -f ./test/wrk/nginx-svc.yaml
  kubectl apply -f ./test/wrk/wrk-deployment.yaml
}

test_case_uninstall() {
  echo "uninstall test case."
  if kubectl get deployment | grep -wq "nginx-deployment"; then
    kubectl delete deployment nginx-deployment
  fi

  if kubectl get svc | grep -wq "nginx-svc"; then
    kubectl delete svc nginx-svc
  fi

  if kubectl get deployment | grep -wq "wrk-deployment"; then
    kubectl delete deployment wrk-deployment
  fi
}

if [ "$1" = "plugin_install" ]; then
  plugin_install
elif [ "$1" = "plugin_uninstall" ]; then
  plugin_uninstall
elif [ "$1" = "test_case_install" ]; then
  test_case_install
elif [ "$1" = "test_case_uninstall" ]; then
  test_case_uninstall
else
  echo "invalid arguement."
fi
