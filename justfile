# Detect if sudo is needed for docker
DOCKER := if docker ps >/dev/null 2>&1; then "docker" else "sudo docker" fi
SUDO   := if docker ps >/dev/null 2>&1; then "" else "sudo" fi

# --- High Level Recipes ---

# Full setup of the environment and images
setup: setup-sysctl setup-python setup-deps setup-zig setup-bazel build-rapidsnark build-prysm build-images setup-kurtosis

# Run the complete simulation cycle
run: start-simulation update-cadvisor-whitelist start-measurements wait-stabilize bootstrap

# --- Detailed Setup Recipes ---

setup-sysctl:
    echo "Configuring sysctl ARP cache limits..."
    @if ! grep -q "net.ipv4.neigh.default.gc_thresh3 = 16384" /etc/sysctl.conf; then \
        echo "Appending ARP cache settings to /etc/sysctl.conf..."; \
        sudo bash -c "cat >> /etc/sysctl.conf <<EOF\n\n# Aetherweave Simulation ARP settings\nnet.ipv4.neigh.default.gc_thresh3 = 16384\nnet.ipv4.neigh.default.gc_thresh2 = 8192\nnet.ipv4.neigh.default.gc_thresh1 = 4096\nEOF"; \
        sudo sysctl -p; \
    fi

setup-python:
    echo "Preparing Python environment..."
    sudo apt-get update && sudo apt-get install -y python3 python3-venv python3-pip
    test -d venv || python3 -m venv venv
    ./venv/bin/pip install -r requirements.txt

setup-deps:
    echo "Installing build dependencies..."
    sudo apt-get install -y cmake git build-essential libssl-dev libgmp-dev libsodium-dev nasm curl m4

setup-zig:
    echo "Installing zig compiler..."
    @if ! [ -x "$(command -v zig)" ]; then \
        curl -L "https://ziglang.org/download/0.15.2/zig-x86_64-linux-0.15.2.tar.xz" -o zig.tar.xz; \
        tar -xf zig.tar.xz; \
        sudo cp zig-x86_64-linux-0.15.2/zig /usr/bin/; \
        sudo cp -r zig-x86_64-linux-0.15.2/lib /usr/lib/zig; \
        rm -rf zig-linux-x86_64-0.15.2 zig.tar.xz; \
    fi

setup-bazel:
    echo "Installing Bazelisk..."
    @if ! [ -x "$(command -v bazel)" ]; then \
        curl -L "https://github.com/bazelbuild/bazelisk/releases/download/v1.28.1/bazelisk-amd64.deb" -o bazelisk.deb; \
        sudo dpkg -i bazelisk.deb; \
        rm bazelisk.deb; \
    fi

build-rapidsnark:
    echo "Building rapidsnark..."
    cd rapidsnark && ./build_gmp.sh host && make host
    echo "Installing rapidsnark libraries in prysm..."
    cp rapidsnark/package/lib/librapidsnark-fr-fq.a prysm/rapidsnark/prover/rapidsnark_vendor/
    cp rapidsnark/package/lib/libgmp.a prysm/rapidsnark/prover/rapidsnark_vendor/libgmp-linux-amd64.a
    cp rapidsnark/package/include/prover.h prysm/rapidsnark/prover/rapidsnark_vendor/

build-prysm:
    echo "Building Prysm..."
    cd prysm && bazel build //cmd/beacon-chain:beacon-chain --config=release
    cp prysm/bazel-bin/cmd/beacon-chain/beacon-chain_/beacon-chain aw_docker/
    chmod +x aw_docker/beacon-chain

setup-docker:
    echo "Checking Docker installation..."
    if ! [ -x "$(command -v docker)" ]; then \
        curl -fsSL https://get.docker.com -o get-docker.sh && sudo sh get-docker.sh && rm get-docker.sh; \
    fi
    if ! groups $USER | grep -q "\bdocker\b"; then \
        sudo usermod -aG docker $USER; \
    fi

setup-kurtosis:
    echo "Installing Kurtosis v1.15.2..."
    @if ! kurtosis version 2>/dev/null | grep -q "1.15.2"; then \
        curl -L "https://github.com/kurtosis-tech/kurtosis-cli-release-artifacts/releases/download/1.15.2/kurtosis-cli_1.15.2_linux_amd64.tar.gz" -o kurtosis.tar.gz; \
        tar -xzf kurtosis.tar.gz; \
        sudo chmod +x kurtosis; \
        sudo mv kurtosis /usr/bin/kurtosis; \
        rm kurtosis.tar.gz; \
    else \
        echo "Kurtosis v1.15.2 already installed."; \
    fi

build-images:
    echo "Loading and building images..."
    {{DOCKER}} load --input cAdvisor/cadvisor_mod_docker_img.tar
    {{DOCKER}} build -t prysm-aetherweave:latest ./aw_docker

# --- Detailed Run Recipes ---

start-simulation:
    echo "Starting Kurtosis simulation..."
    {{SUDO}} kurtosis run --enclave aetherweave ./ethereum-package-4.6.0 --args-file aetherweave-testnet-generated_32.yaml

update-cadvisor-whitelist:
    echo "Updating cadvisor whitelist with dynamic container IDs..."
    $(eval CL_IDS := $(shell {{DOCKER}} ps --format '{{.ID}} {{.Names}}' | grep cl- | tail -n 16 | head -n 15 | awk '{printf "%s%s", (NR>1?",":""), $$1} END {print ""}'))
    sed -i "s/--docker_id_prefix_whitelist=.*/--docker_id_prefix_whitelist={{CL_IDS}}/" measurements/docker-compose.yml

start-measurements:
    echo "Starting measurement containers..."
    cd measurements && {{DOCKER}} compose up -d

wait-stabilize:
    echo "Waiting 20 minutes for stabilization..."
    sleep 1200

bootstrap:
    echo "Bootstrapping simulation..."
    {{SUDO}} ./venv/bin/python3 bootstrap_kurtosis.py

# --- Cleanup ---

clean-simulation:
    {{SUDO}} kurtosis enclave rm -f aetherweave

stop-measurements:
    cd measurements && {{DOCKER}} compose down
