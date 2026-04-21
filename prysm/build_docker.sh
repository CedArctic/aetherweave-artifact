# bazel build //cmd/beacon-chain:push_images --config=release
bazel build //cmd/beacon-chain:beacon-chain --config=release
cp ./bazel-bin/cmd/beacon-chain/beacon-chain_/beacon-chain simulations/aw_docker/
cd ./simulations/aw_docker
docker build -t prysm-aetherweave:latest .
rm beacon-chain