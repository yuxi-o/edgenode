# SPDX-License-Identifier: Apache-2.0
# Copyright (c) 2019 Intel Corporation

export GO111MODULE = on
export GOPROXY = https://goproxy.cn

.PHONY: build appliance eaa interfaceservice edgedns clean build-docker lint test help build-docker-hddl hddllog build-docker-fpga-cfg build-docker-fpga-opae
TMP_DIR:=$(shell mktemp -d)
BUILD_DIR ?=dist

VER:=1.0

ifeq ($(KUBE_OVN_MODE), True)
build: eaa interfaceservice edgedns
else
build: edalibs appliance eaa edgedns nts
endif

appliance:
	mkdir -p "${BUILD_DIR}"
	GOOS=linux go build -o "${BUILD_DIR}/appliance" ./cmd/appliance

eaa:
	mkdir -p "${BUILD_DIR}"
	GOOS=linux go build -o "${BUILD_DIR}/eaa" ./cmd/eaa

interfaceservice:
	mkdir -p "${BUILD_DIR}"
	GOOS=linux go build -o "${BUILD_DIR}/interfaceservice" ./cmd/interfaceservice

edgedns:
	mkdir -p "${BUILD_DIR}"
	GOOS=linux go build -a --ldflags '-extldflags "-static"' -tags netgo -installsuffix netgo -o "${BUILD_DIR}/edgednssvr" ./cmd/edgednssvr

nts:
	make -C internal/nts

edalibs:
	make -C internal/nts/eda_libs

hddllog:
	mkdir -p "${BUILD_DIR}"
	GOOS=linux go build -o "${BUILD_DIR}/hddllog" ./cmd/hddllog

clean:
	rm -rf "${BUILD_DIR}"
	make clean -C internal/nts
	make clean -C internal/nts/eda_libs

build-docker: build
	cp docker-compose.yml "${TMP_DIR}"
	cp build/eaa/Dockerfile "${TMP_DIR}/Dockerfile_eaa"
	cp build/eaa/entrypoint_eaa.sh "${TMP_DIR}"
	cp "${BUILD_DIR}/eaa" "${TMP_DIR}"
	cp build/edgednssvr/Dockerfile "${TMP_DIR}/Dockerfile_edgednssvr"
	cp "${BUILD_DIR}/edgednssvr" "${TMP_DIR}"
ifeq ($(KUBE_OVN_MODE), True)
	cp build/interfaceservice/Dockerfile "${TMP_DIR}/Dockerfile_interfaceservice"
	cp build/interfaceservice/entrypoint_interfaceservice.sh "${TMP_DIR}"
	cp "${BUILD_DIR}/interfaceservice" "${TMP_DIR}"
	cd "${TMP_DIR}" && VER=${VER} docker-compose build eaa interfaceservice edgednssvr syslog-ng
else
	cp build/appliance/Dockerfile "${TMP_DIR}/Dockerfile_appliance"
	cp build/appliance/entrypoint.sh "${TMP_DIR}"
	cp /opt/dpdk-18.11.2/usertools/dpdk-devbind.py "${TMP_DIR}"
	cp "${BUILD_DIR}/appliance" "${TMP_DIR}"
	mkdir -p "${TMP_DIR}/nts"
	cp internal/nts/build/nes-daemon "${TMP_DIR}/nts"
	cp internal/nts/kni_docker_daemon.py "${TMP_DIR}/nts"
	cp internal/nts/ovs_docker_daemon.py "${TMP_DIR}/nts"
	cp internal/nts/entrypoint.sh "${TMP_DIR}/nts"
	cp internal/nts/build/libnes_api_shared.so "${TMP_DIR}/nts"
	cp internal/nts/Dockerfile "${TMP_DIR}/Dockerfile_nts"
	cd "${TMP_DIR}" && VER=${VER} docker-compose build appliance nts eaa edgednssvr syslog-ng
endif
	ls "${TMP_DIR}"
	rm -rf "${TMP_DIR}"

build-docker-hddl: hddllog
	cp build/hddlservice/Dockerfile "${TMP_DIR}/Dockerfile_hddlservice"
	cp build/hddlservice/start.sh "${TMP_DIR}"
	cp build/hddlservice/docker-compose.yml "${TMP_DIR}"
	cp "${BUILD_DIR}/hddllog" "${TMP_DIR}"
	cd "${TMP_DIR}" && VER=${VER} docker-compose build
	ls "${TMP_DIR}"
	rm -rf "${TMP_DIR}"

build-docker-biosfw:
	cp build/biosfw/Dockerfile "${TMP_DIR}/Dockerfile_biosfw"
	cp build/biosfw/biosfw.sh "${TMP_DIR}"
	cp build/biosfw/syscfg_package.zip "${TMP_DIR}"
	cd "${TMP_DIR}" && docker build -t openness-biosfw -f Dockerfile_biosfw .
	rm -rf "${TMP_DIR}"

build-docker-fpga-cfg:
	cp build/fpga_config/Dockerfile "${TMP_DIR}/Dockerfile_fpga"
	cp -r build/fpga_config/bbdev_config_service "${TMP_DIR}"
	cd "${TMP_DIR}" && docker build -t fpga-config-utility:1.0 -f Dockerfile_fpga .
	rm -rf "${TMP_DIR}"

build-docker-fpga-opae:
	cp build/fpga_opae/Dockerfile "${TMP_DIR}/Dockerfile_opae"
	cp -r build/fpga_opae/n3000-1-3-5-beta-cfg-2x2x25g-setup.zip "${TMP_DIR}"
	cp -r build/fpga_opae/n3000-1-3-5-beta-rte-setup.zip "${TMP_DIR}"
	cp -r build/fpga_opae/expect_script.sh "${TMP_DIR}"
	cp -r build/fpga_opae/check_if_modules_loaded.sh "${TMP_DIR}"
	cd "${TMP_DIR}" && docker build -t fpga-opae-pacn3000:1.0 -f Dockerfile_opae .
	rm -rf "${TMP_DIR}"

run-docker:
ifeq ($(KUBE_OVN_MODE), False)
	VER=${VER} docker-compose up appliance nts eaa edgednssvr syslog-ng --no-build
endif

lint: edalibs
	golangci-lint run

test: edalibs
	http_proxy= https_proxy= HTTP_PROXY= HTTPS_PROXY= ginkgo -v -r --randomizeSuites --failOnPending --skipPackage=vendor,edants,wrappers,stubs

help:
	@echo "Please use \`make <target>\` where <target> is one of"
	@echo "  build                  to build the appliance application, EAA, edgedns server and NTS"
	@echo "  appliance              to build the appliance application"
	@echo "  interfaceservice       to build the interfaceservice"
	@echo "  eaa                    to build the EAA"
	@echo "  edgedns                to build the edgedns server"
	@echo "  nts                    to build the NTS"
	@echo "  hddllog                to build the log supporting hddl service"
	@echo "  clean                  to clean up build artifacts and docker"
	@echo "  build-docker           to build the release docker image"
	@echo "  build-docker-hddl      to build optional docker image for hddl-service"
	@echo "  build-docker-biosfw    to build optional docker image for biosfw feature"
	@echo "  build-docker-fpga-cfg  to build optional docker image for bbdev configuration utility"
	@echo "  build-docker-fpga-opae to build optional docker image for opae"
	@echo "  run-docker             to start containers"
	@echo "  lint                   to run linters and static analysis on the code"
	@echo "  test                   to run unit tests"
