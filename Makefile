IMG ?= hongxuming/nriplugin:0.1

CONTAINER_TOOL ?= nerdctl

#----------------------------------------#
#       镜像构建、推送、打包             #
#----------------------------------------#
.PHONY: docker-build #制作NRI插件镜像
docker-build:
	${CONTAINER_TOOL} build -t ${IMG} . -f Dockerfile

.PHONY: docker-push #推送NRI插件到仓库
docker-push:
	${CONTAINER_TOOL} push ${IMG}

.PHONY: pack # 打包NRI插件为tar包
pack:
	make docker-build
	mkdir -p bin
	${CONTAINER_TOOL} save -o bin/nriplugin.tar ${IMG}

#----------------------------------------#
#         构建gprc服务器                 #
#----------------------------------------#
GRPC_SERVER_BUILD_DIR = pkg/podAffinityServer
OBJECT = grpc_server
OBJECT_DIR = bin
.PHONY: build_grpc_server
build_grpc_server: $(GRPC_SERVER_BUILD_DIR)/numafast.c $(GRPC_SERVER_BUILD_DIR)/numafast.o
	ar rcs $(GRPC_SERVER_BUILD_DIR)/libnumafast.a $(GRPC_SERVER_BUILD_DIR)/numafast.o
	go build -o ${OBJECT_DIR}/${OBJECT} cmd/grpcServer/main.go

$(GRPC_SERVER_BUILD_DIR)/numafast.o: $(GRPC_SERVER_BUILD_DIR)/numafast.c
	gcc -c -o $(GRPC_SERVER_BUILD_DIR)/numafast.o ${GRPC_SERVER_BUILD_DIR}/numafast.c

.PHONY: grpc_clean
grpc_clean:
	rm -f ${GRPC_SERVER_BUILD_DIR}/libnumafast.a ${GRPC_SERVER_BUILD_DIR}/numafast.o

#----------------------------------------#
#         NRI插件及测试用例的部署        #
#----------------------------------------#
.PHONY: plugin_install #安装NRI插件
plugin_install:
	chmod +x deploy.sh
	./deploy.sh "plugin_install"

.PHONY: plugin_uninstall #卸载NRI插件
plugin_uninstall:
	chmod +x deploy.sh
	./deploy.sh "plugin_uninstall"

.PHONY: test_case_install #安装测试用例
test_case_install:
	chmod +x deploy.sh
	./deploy.sh "test_case_install"

.PHONY: test_case_uninstall
test_case_uninstall:
	chmod +x deploy.sh
	./deploy.sh "test_case_uninstall"

#----------------------------------------#
#          编译NRI插件二进制文件         #
#----------------------------------------#
.PHONY: build_binary
build_binary:
	mkdir -p bin
	go build -o bin/nriplugin cmd/nriplugin/main.go
