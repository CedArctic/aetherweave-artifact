#!/bin/bash

# Exit on error
set -e

echo "--- Starting Aetherweave Simulation Setup ---"

# 0. Configure sysctl for large scale simulation
echo "Step 0: Configuring sysctl ARP cache limits..."
SYSCTL_CONF="/etc/sysctl.conf"
if ! grep -q "net.ipv4.neigh.default.gc_thresh3 = 16384" "$SYSCTL_CONF"; then
    echo "Appending ARP cache settings to $SYSCTL_CONF..."
    sudo bash -c "cat >> $SYSCTL_CONF <<EOF

# Aetherweave Simulation ARP settings
net.ipv4.neigh.default.gc_thresh3 = 16384
net.ipv4.neigh.default.gc_thresh2 = 8192
net.ipv4.neigh.default.gc_thresh1 = 4096

# Increase open files count limit
fs.inotify.max_user_instances = 1024
EOF"
    sudo sysctl -p
fi

# 0.5. Initialize submodules
echo "Step 0.5: Initializing submodules..."
git submodule update --init --recursive

# 1. Install python and prepare venv
echo "Step 1: Preparing Python environment..."
sudo apt-get update
sudo apt-get install -y python3 python3-venv python3-pip
if [ ! -d "venv" ]; then
    python3 -m venv venv
fi
source venv/bin/activate
pip install -r requirements.txt

# 2. Install Prysm dependencies
echo "Step 2: Installing Prysm and rapidsnark build dependencies"
sudo apt-get install -y cmake git build-essential libssl-dev libgmp-dev libsodium-dev nasm curl m4

# 3. Install zig
echo "Step 3: Installing zig compiler"
ZIG_VERSION="0.15.2"
ZIG_URL="https://ziglang.org/download/${ZIG_VERSION}/zig-x86_64-linux-${ZIG_VERSION}.tar.xz"
ZIG_FOLDER="zig-x86_64-linux-${ZIG_VERSION}"

if ! [ -x "$(command -v zig)" ]; then
    curl -L "${ZIG_URL}" -o zig.tar.xz
    tar -xf zig.tar.xz
    sudo cp "${ZIG_FOLDER}/zig" /usr/bin/
    sudo cp -r "${ZIG_FOLDER}/lib" /usr/lib/zig
    rm -rf "${ZIG_FOLDER}" zig.tar.xz
fi

# 4. Install bazelisk
echo "Step 4: Installing bazel"
BAZELISK_VERSION="1.28.1"
BAZELISK_URL="https://github.com/bazelbuild/bazelisk/releases/download/v${BAZELISK_VERSION}/bazelisk-amd64.deb"

if ! [ -x "$(command -v bazel)" ]; then
    echo "Installing Bazelisk..."
    curl -L "${BAZELISK_URL}" -o bazelisk.deb
    sudo dpkg -i bazelisk.deb
    rm bazelisk.deb
fi

# 5. Install docker and enable running containers without sudo
echo "Step 5: Checking Docker installation..."
if ! [ -x "$(command -v docker)" ]; then
    echo "Installing Docker..."
    curl -fsSL https://get.docker.com -o get-docker.sh
    sudo sh get-docker.sh
    rm get-docker.sh
fi

# Check if user is in docker group
if ! groups $USER | grep &>/dev/null "\bdocker\b"; then
    echo "Adding user to docker group..."
    sudo usermod -aG docker $USER
    echo "NOTE: Group changes require a fresh session. Fallback to sudo for now."
fi

# Detect if we need sudo for docker/kurtosis commands in this session
DOCKER="docker"
SUDO_IF_NEEDED=""
if ! docker ps >/dev/null 2>&1; then
    DOCKER="sudo docker"
    SUDO_IF_NEEDED="sudo"
fi

# 6. Load the custom cadvisor image
echo "Step 6: Loading custom cadvisor image..."
$DOCKER load --input cAdvisor/cadvisor_mod_docker_img.tar

# 7. Build rapidsnark using zig
echo "Step 7: Building rapidsnark with zig"
(cd rapidsnark && make clean && ./build_gmp.sh host && make host)

# Overwrite vendored rapidsnark libraries in prysm
echo "Installing rapidsnark libraries in prysm build directory"
cp rapidsnark/package/lib/librapidsnark-fr-fq.a prysm/rapidsnark/prover/rapidsnark_vendor/
cp rapidsnark/package/lib/libgmp.a prysm/rapidsnark/prover/rapidsnark_vendor/libgmp-linux-amd64.a
cp rapidsnark/package/include/prover.h prysm/rapidsnark/prover/rapidsnark_vendor/

# 8. Build Prysm
echo "Step 8: Building Prysm"
(cd prysm && bazel build //cmd/beacon-chain:beacon-chain --config=release)
echo "Copying beacon-chain executable to docker image directory"
# Using standard bazel-bin path
cp prysm/bazel-bin/cmd/beacon-chain/beacon-chain_/beacon-chain aw_docker/
chmod +x aw_docker/beacon-chain

# 9. Build the docker image in aw_docker/
echo "Step 9: Building Aetherweave Beacon Chain image..."
$DOCKER build -t prysm-aetherweave:latest ./aw_docker

# 10. Install the kurtosis binary (Version 1.15.2)
echo "Step 10: Installing Kurtosis v1.15.2..."
KURTOSIS_VERSION="1.15.2"
KURTOSIS_URL="https://github.com/kurtosis-tech/kurtosis-cli-release-artifacts/releases/download/${KURTOSIS_VERSION}/kurtosis-cli_${KURTOSIS_VERSION}_linux_amd64.tar.gz"

if ! kurtosis version 2>/dev/null | grep -q "${KURTOSIS_VERSION}"; then
    echo "Downloading Kurtosis v${KURTOSIS_VERSION}..."
    curl -L "${KURTOSIS_URL}" -o kurtosis.tar.gz
    tar -xzf kurtosis.tar.gz
    sudo chmod +x kurtosis
    sudo mv kurtosis /usr/bin/kurtosis
    rm kurtosis.tar.gz
    echo "Kurtosis v${KURTOSIS_VERSION} installed to /usr/bin/kurtosis."
else
    echo "Kurtosis v${KURTOSIS_VERSION} is already installed."
fi

# 11. Start the simulation
echo "Step 11: Starting Kurtosis simulation..."
$SUDO_IF_NEEDED kurtosis run --enclave aetherweave ./ethereum-package-4.6.0 --args-file aetherweave-testnet-generated_32.yaml

echo "Getting the docker IDs of a sample of the consensus nodes"
CL_IDS=$($DOCKER ps --format '{{.ID}} {{.Names}}' | grep cl- | tail -n 16 | head -n 15 | awk '{printf "%s%s", (NR>1?",":""), $1} END {print ""}')

echo "Overwriting docker IDs in measurements/docker-compose.yml"
sed -i "s/--docker_id_prefix_whitelist=.*/--docker_id_prefix_whitelist=$CL_IDS/" measurements/docker-compose.yml

# 12. Start the measurement containers
echo "Step 12: Starting measurement containers..."
(cd measurements && $DOCKER compose up -d)

# 13. Wait for 20 minutes for stabilization
echo "Step 13: Waiting 20 minutes for simulation to stabilize (stake deposits)..."
sleep 1200

# 14. Run bootstrap_kurtosis.py
echo "Step 14: Bootstrapping simulation and starting loggers..."
source venv/bin/activate
$SUDO_IF_NEEDED python3 bootstrap_kurtosis.py

echo "--- Setup and Bootstrap Complete ---"
