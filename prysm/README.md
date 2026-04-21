# Modified Prysm for stake-based peer discovery

## Build
1. Prepare build host with [official Prysm instructions](https://prysm.offchainlabs.com/docs/install-prysm/install-with-bazel/)
2. Build with `build.sh`

## Simulation
1. Download and install Kurtosis
2. Download ethereum-package v4.6.0
3. Build the docker container using `build_docker.sh`
4. Run the simulation using the provided configuration file: `kurtosis run --enclave aetherweave ./path/to/ethereum-package-4.6.0 --args-file aetherweave-testnet-generated_32.yaml`
5. Wait for simulation to start, and then get the container IDs of some of the modified client containers: `docker ps --format 'table {{.ID}} {{.Names}} {{.Image}} | grep cl- | tail -n 16 | awk '{ print $1 }'`
6. Add container IDs to `simulations/benchmarks/docker-compose.yml` if using our cAdvisor fork that supports docker container ID whitelisting
7. Navigate to `simulations/benchmarks` in the terminal and start the monitoring stack: `docker compose up -d`
8. Monitor client logs until all clients succeed in depositing stake: `kurtosis service logs -f aetherweave cl-001-...`
9. Run bootstraping script: `simulations/bootstrap_kurtosis.py`

## Files
### Protocol
- `beacon-chain/sync/aetherweave.go`: Main protocol implementation
- `beacon-chain/sync/aetherweave_zk.go`: ZKP-related functionality
- `beacon-chain/sync/rpc_aetherweave.go`: RPC related functionality
- `proto/prysm/v1alpha1/aetherweave.proto`: Protobuf definition of protocol messages
- `proto/prysm/v1alpha1/aetherweave.pb.go`: Protobuf go bindings. Generated from `aetherweave.proto`
- `beacon-chain/sync/awcontract/aetherweaveprivate.go`: Staking contract go bindings. Generated with abigen from smart contract.
- Test files:
    - `beacon-chain/sync/aetherweave_test.go`
    - `beacon-chain/sync/aetherweave_zk_test.go`
    - `beacon-chain/sync/rpc_aetherweave_test.go`
    - `beacon-chain/sync/testdata/`: Files for testing ZKP functionality.

### Simulation
- `simulations/aw_docker`: Files for building the modified prysm docker container.
- `simulations/benchmarks`: Files for setting up the cAdvisor + Prometheus + Grafana monitoring stack after the experiment has started
- `simulations/aetherweave-testnet-generated_32.yaml`: Configuration file for the simulation. Use with ethereum-package v4.6.0, and Kurtosis.
- `simulations/bootstrap_kurtosis.py`: Bootstrap script for the protocol. After clients have successfully deposited stake, run this to start loggers and exchange NetworkRecords (.nr files) between clients.
- `simulations/generate_config.py`: Simulation configuration file generator.

### Build Scripts
- `build.sh`: Build the modified client
- `build_docker.sh`: Build the client and create a docker image for use in the simulation.
- `build_aw_protobufs.sh`: Build and copy the protocol's protobufs.

### Staking contract and ZKPs
- `smart_contract/circuits/`: Stake and Share proof ZKP circuits
- `smart_contract/contracts/AetherWeavePrivate.sol`: Staking contract
- `smart_contract/justfile`: Use to compile circuits, contract, and generate contract go bindings.

### Other
- `beacon-chain/sync/service.go`: Prysm Sync service file. Modified to start the protocol scheduler.
- `wasmer/`: Version of wasmer needed for linking with rapidsnark.
- `rapidsnark/`: Copy of custom-compiled rapidsnark. We use zig-compiled binaries for the rapidsnark prover.


## License
prysm is released under the GPL v3 license. wasmer is available under the MIT license. go-rapidsnark is available under either the MIT or Apache v2 license.