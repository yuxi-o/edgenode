# SPDX-License-Identifier: Apache-2.0
# Copyright (c) 2019 Intel Corporation

FROM centos:7.6.1810 AS nts-deps-image

RUN mkdir ~/.pip && \
    echo -e "[global]\nindex-url = http://mirrors.aliyun.com/pypi/simple/\ntimeout = 6000\n\n[install]\ntrusted-host = mirrors.aliyun.com" >> ~/.pip/pip.conf

RUN rm -rf /etc/yum.repos.d/*
RUN cd /etc/yum.repos.d && curl -o CentOS-Base.repo "http://mirrors.aliyun.com/repo/Centos-7.repo"
RUN cd /etc/yum.repos.d && curl -o epel.repo "http://mirrors.aliyun.com/repo/epel-7.repo"
RUN yum clean all

RUN yum upgrade -y ca-certificates && \
    yum install -y epel-release && \
    yum install -y numactl-devel libhugetlbfs-utils iproute python python-pip sudo && \
    pip install docker && \
    pip install configparser

FROM nts-deps-image

ARG username=nts
ARG user_dir=/home/$username

RUN useradd -d $user_dir -m -s /bin/bash $username
RUN groupadd sudo
RUN usermod -aG sudo $username
RUN echo '%sudo ALL=(ALL) NOPASSWD:ALL' >> /etc/sudoers

USER $username
WORKDIR $user_dir

COPY ./nts/nes-daemon ./
COPY ./nts/kni_docker_daemon.py ./
COPY ./nts/ovs_docker_daemon.py ./
COPY ./nts/entrypoint.sh ./
COPY ./nts/libnes_api_shared.so ./

ENTRYPOINT ["sudo", "-E", "./entrypoint.sh"]
