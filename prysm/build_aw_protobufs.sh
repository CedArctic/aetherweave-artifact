#!/usr/bin/bash

# Build all protobufs
bazel build //proto/prysm/v1alpha1:*

# Copy over files related to Aetherweave
sudo cp ./bazel-bin/proto/prysm/v1alpha1/go_proto_/github.com/OffchainLabs/prysm/v6/proto/prysm/v1alpha1/aetherweave.pb.go ./proto/prysm/v1alpha1/

sudo cp ./bazel-bin/proto/prysm/v1alpha1/phase0.ssz.go ./proto/prysm/v1alpha1/