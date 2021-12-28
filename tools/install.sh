#!/bin/bash
BIN_DIR=/usr/bin
CUR_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

DOCKER_CONF=/etc/docker/daemon.json
CONTAINERD_CONF=/etc/containerd/config.toml

rpms=(
  libnvidia-container1-1.3.3-1.x86_64
  libnvidia-container-tools-1.3.3-1.x86_64
  nvidia-container-toolkit-1.4.2-2.x86_64
  nvidia-container-runtime-3.4.2-1.x86_64
)

debs=(
  libnvidia-container1_1.3.3-1_amd64
  libnvidia-container-tools_1.3.3-1_amd64
  nvidia-container-toolkit_1.4.2-1_amd64
  nvidia-container-runtime_3.4.2-1_amd64
)



gpu::nv_runtime::installer() {
  OS_RELEASE="$(. /etc/os-release && echo "$ID")"
  echo "OS_RELEASE=$OS_RELEASE"
  cd ${CUR_DIR}

  if [ "tlinux" == "$OS_RELEASE" ] || [ "centos" == "$OS_RELEASE" ] || [ "tencentos" == "$OS_RELEASE" ]; then
    len=${#rpms[*]}
    for ((i = 0; i < len; i++)); do
      if rpm -q ${rpms[i]} >/dev/null 2>&1; then
        echo "${rpms[i]} already installed"
      else
        rpm -i ${CUR_DIR}/${rpms[i]}.rpm
      fi
    done
  elif [ "ubuntu" == "$OS_RELEASE" ]; then
    len=${#debs[*]}
    for ((i = 0; i < len; i++)); do
      if dpkg -s ${debs[i]%%_*} >/dev/null 2>&1; then
        echo "${debs[i]} already installed"
      else
        dpkg -i ${CUR_DIR}/${debs[i]}.deb
      fi
    done
  fi
}

gpu::docker::installer() {
  # 1. add nvidia-docker
  mv ${CUR_DIR}/nvidia-docker ${BIN_DIR}

  # 2. patch and install config
  ${CUR_DIR}/conf-patch -o ${DOCKER_CONF} -p ${CUR_DIR}/docker-gpu.json >${CUR_DIR}/daemon.json

  # 3. mv daemon.json
  mv -f ${CUR_DIR}/daemon.json ${DOCKER_CONF}
}

gpu::containerd::installer() {
  ${CUR_DIR}/conf-patch -o ${CONTAINERD_CONF} -p ${CUR_DIR}/containerd-gpu.toml >${CUR_DIR}/config.toml

  mv -f ${CUR_DIR}/config.toml ${CONTAINERD_CONF}
}

gpu::main() {
  if [ -d "/sys/class/drm/card1/device" ]; then
	dev_id=$(cat /sys/class/drm/card1/device/device)
	vendor_id=$(cat /sys/class/drm/card1/device/vendor)
	if [ "$dev_id" == "0x4907" ] && [ "$vendor_id" == "0x8086" ]; then
      echo "intel GPU"
      exit 0
	fi
  fi
  if [ -d "/sys/class/drm/card0/device" ]; then
	dev_id=$(cat /sys/class/drm/card0/device/device)
	vendor_id=$(cat /sys/class/drm/card0/device/vendor)
	if [ "$dev_id" == "0x4907" ] && [ "$vendor_id" == "0x8086" ]; then
      echo "intel GPU"
      exit 0
	fi
  fi

  echo "nvidia GPU"
  gpu::nv_driver::installer
  gpu::nv_runtime::installer
  if type docker >/dev/null 2>&1; then
    gpu::docker::installer
  else
    gpu::containerd::installer
  fi
}

gpu::main















#set -x
#export PATH=${PATH}:/host/usr/bin/:/host/usr/sbin
#if ! /usr/bin/installer --endpoint "$1"; then
#    exit 1
#fi

#mv /host/usr/bin/cgroupfs-container-runtime-hook   /host/usr/bin/cgroupfs-container-runtime-hook -bak
#mv /host/usr/bin/nvidia-container-toolkit       /host/usr/bin/nvidia-container-toolkit-bak
#mv /host/usr/bin/mount_cgroup                     /host/usr/bin/mount_cgroup-bak
#cp /usr/bin/cgroupfs-container-runtime-hook     /host/usr/bin/cgroupfs-container-runtime-hook
#cp /usr/bin/qgpu-nvidia-container-toolkit       /host/usr/bin/nvidia-container-toolkit
#cp /usr/bin/mount_cgroup                          /host/usr/bin/mount_cgroup