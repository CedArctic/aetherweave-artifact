# AetherWeave: Stake-Backed Peer Discovery Protocol

## Artifact Structure

- `prysm/`: Modified Prysm consensus client implementing the AetherWeave protocol.
- `rapidsnark/`: Zero-knowledge proof (ZKP) prover used for generating stake-backed identity proofs.
- `aw_docker/`: Docker image for the AetherWeave beacon chain nodes.
- `ethereum-package-4.6.0/`: Kurtosis package used to orchestrate the local testnet simulation.
- `measurements/`: Monitoring stack (Prometheus, Grafana, cAdvisor).
- `setup_simulation.sh`: Monolithic script to automate environment setup and simulation.
- `justfile`: Modular task runner for separate setup and execution steps.
- `bootstrap_kurtosis.py`: Post-launch script to bootstrap peer records and manage logging.

## Protocol implementation code structure
See `prysm/README.md`.

## Prerequisites

- **Operating System**: Linux (Ubuntu 22.04+ recommended).
- **Hardware**: Recommended 64 logical core CPU for 100-node simulation
- **Dependencies**: `curl`, `git`, `sudo` access.

## Quick Start

The simulation is preconfigured with 100 AetherWeave container nodes. The scheduler in `prysm/beacon-chain/sync/aetherweave.go: aetherweaveScheduler()` periodically cycles through table sizes that effectively simulate networks of 100, 225, 400 and 625 nodes.

### 1. Automatic Setup
You can use the provided `justfile` (requires `just`) or the bash script.

**Using Just (Recommended):**
```bash
# 1. Prepare system dependencies, ZKP libraries, and build the Prysm binary
just setup

# 2. Start the simulation, monitoring, and bootstrapping
just run
```

**Using Bash:**
```bash
chmod +x setup_simulation.sh
./setup_simulation.sh
```

### 2. Manual/Modular Steps
To run steps individually:
- `just setup-sysctl`: Configure ARP cache limits and open files limit to facilitate simulation.
- `just build-prysm`: Compile the modified beacon chain client.
- `just start-simulation`: Launch the Kurtosis enclave.
- `just bootstrap`: Initial exchange of network records between AetherWeave clients to simulate network bootstrapping and start log collection.

---

## Evaluating Results

Results are delivered through three primary channels:

### 1. Grafana Dashboard (Visual Metrics)
Real-time visualization of container performance and network behavior.
- **URL**: `http://localhost:3000`
- **Credentials**: `admin` / `admin`
- **Dashboard**: "Cadvisor exporter"
- **Key Metrics**: CPU/Memory/Network utilization of beacon clients.

### 2. cAdvisor (Real-time Container Stats)
Alternative container performance metrics visualization.
- **URL**: `http://localhost:8080`

### 3. Log Files (Protocol Behavior)
The `bootstrap_kurtosis.py` script automatically spawns loggers for all 100 consensus nodes.
- **Location**: `logs/<TIMESTAMP>/`
- **Format**: `cl-<node_id>-<timestamp>.log`
- **What to look for**: Stake deposit, hearthbeat, record request and response events.


## Configuration
- Customize the simulation nodes in `aetherweave-testnet-generated_32.yaml`.
- Customize protocol parameters and the scheduler in `prysm/beacon-chain/sync/aetherweave.go`.

## Licenses
Components bundled in this repository come with their own respective licenses. Our code is licensed under the MIT license. prysm is released under the GPL v3 license. wasmer is available under the MIT license. go-rapidsnark is available under either the MIT or Apache v2 license. cAdvisor is licensed under the Apache v2 license.
