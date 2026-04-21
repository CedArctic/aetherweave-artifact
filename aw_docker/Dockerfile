# Use the gcr.io/offchainlabs/prysm/beacon-chain:latest image as the base
FROM gcr.io/offchainlabs/prysm/beacon-chain:latest

# Set the working directory
WORKDIR /

# Copy the Aetherweave beacon-chain binary into the root of the image,
# overwriting the existing one.
COPY beacon-chain /beacon-chain

# Copy the circom files
ADD zk /root/.eth2

# Ensure the prysm binary has executable permissions
RUN chmod +x /beacon-chain

# Install libwasmer.so
# Source: https://github.com/iden3/wasmer-go/tree/master/wasmer/packaged/lib
COPY libwasmer.so /lib/x86_64-linux-gnu/libwasmer.so
