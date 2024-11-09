IMG ?= hongxuming/wrk:latest
wrk-build:
	nerdctl build -t ${IMG} . -f Dockerfile.wrk_test

wrk-push:
	nerdctl push ${IMG}

wrk-install:
	kubectl delete deploy wrk-deployment
	kubectl apply -f wrk-deployment.yaml

all:
	make wrk-build
	make wrk-push
	make wrk-install
