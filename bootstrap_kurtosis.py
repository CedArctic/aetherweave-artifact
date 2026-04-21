"""
Python script that starts loggers for the consensus client containers, and bootstraps the network records in their tables.
"""

import io
import json
import os
import tarfile
import threading
import time

import docker

# Number of CL nodes
CL_NODES = 100

# Number of peers with which to bootstrap nodes
BOOTSTRAP_PEERS_CT = 100

# Directory to place network record files
LOGTIME = time.time()
NR_DIR = f"nr_files/{LOGTIME}"

# Logs directory
LOGS_DIR = f"logs/{LOGTIME}"


# Function to find the full container name using a prefix
def find_container_by_prefix(client, prefix):
    containers = client.containers.list(all=True)
    for container in containers:
        if container.name.startswith(prefix):
            return container.name
    return None


def log_container(container_name, log_file_path):
    """
    Continuously logs the output of a docker container to a file.
    """
    client = docker.from_env()
    try:
        container = client.containers.get(container_name)
        with open(log_file_path, "a") as f:
            for line in container.logs(stream=True, follow=True):
                f.write(line.decode("utf-8"))
    except docker.errors.NotFound:
        print(f"Container {container_name} not found.")
    except Exception as e:
        print(f"An error occurred while logging {container_name}: {e}")


def copy_file_from_container(container_name, src_path, dest_path):
    """
    Copies a file from a container to the local filesystem using get_archive.
    """
    client = docker.from_env()
    try:
        container = client.containers.get(container_name)

        # Get the tar archive of the file
        strm, stat = container.get_archive(src_path)

        # Read the generator into a single bytes object
        file_bytes = b"".join(strm)
        file_obj = io.BytesIO(file_bytes)

        # Extract the file from the tar archive
        with tarfile.open(fileobj=file_obj, mode="r") as tar:
            # The file is expected to be at the root of the archive
            member_name = os.path.basename(src_path)
            tar.extract(member_name, os.path.dirname(dest_path))

            # Rename the extracted file to the desired destination path
            os.rename(os.path.join(os.path.dirname(dest_path), member_name), dest_path)

        print(f"Copied {src_path} from {container_name} to {dest_path}")
    except docker.errors.NotFound:
        print(f"Container {container_name} not found.")
    except Exception as e:
        print(f"An error occurred: {e}")


def copy_file_to_container(container_name, src_path, dest_path):
    """
    Copies a file from the local filesystem into a container using a temporary tar archive.
    """
    client = docker.from_env()
    try:
        container = client.containers.get(container_name)

        # Create a temporary tar archive of the file
        tar_filename = "temp.tar"
        os.system(
            f"tar -czf {tar_filename} -C {os.path.dirname(src_path)} {os.path.basename(src_path)}"
        )

        with open(tar_filename, "rb") as f:
            container.put_archive(os.path.dirname(dest_path), f.read())

        os.remove(tar_filename)
        print(f"Copied {src_path} to {container_name}:{dest_path}")
    except docker.errors.NotFound:
        print(f"Container {container_name} not found.")
    except Exception as e:
        print(f"An error occurred: {e}")


def main():
    """
    Main function to orchestrate all the tasks.
    """

    client = docker.from_env()

    # 1. Start logging threads for each container
    print("Starting loggers...")
    if not os.path.exists(LOGS_DIR):
        os.makedirs(LOGS_DIR)
    log_threads = []
    container_map = {}
    for container_num in range(1, CL_NODES + 1):
        # Add padding to name
        c_name = str(container_num)
        if CL_NODES >= 10 and container_num < 10:
            c_name = "0" + c_name
        if CL_NODES >= 100 and container_num < 100:
            c_name = "0" + c_name
        prefix = f"cl-{c_name}-prysm-geth--"
        container_name = find_container_by_prefix(client, prefix)
        if container_name:
            container_map[c_name] = container_name
            log_file_path = f"{LOGS_DIR}/cl-{c_name}-{LOGTIME}.log"
            thread = threading.Thread(
                target=log_container, args=(container_name, log_file_path)
            )
            thread.daemon = True
            thread.start()
            log_threads.append(thread)
            print(f"Logger for {container_name} started.")
        else:
            print(f"Container with prefix {prefix} not found.")

    time.sleep(2)

    # 2. Copy .nr files from containers upon initialization
    print("\nCopying .nr files from containers...")
    if not os.path.exists(NR_DIR):
        os.makedirs(NR_DIR)

    for container_num, container_name in container_map.items():
        src_nr_path = "/root/.eth2/local_NetworkRecord.nr"
        dest_nr_path = f"{NR_DIR}/cl-{container_num}.nr"
        copy_file_from_container(container_name, src_nr_path, dest_nr_path)

    # 3. Copy specified .nr files into containers
    print("\nImporting .nr files into containers...")

    # Using all-to-all
    for dst_num, dst_name in container_map.items():
        for src_num, _ in container_map.items():
            # if dst_num != src_num:
            if int(dst_num) + BOOTSTRAP_PEERS_CT >= int(src_num) and int(dst_num) < int(
                src_num
            ):
                src_nr_path = f"{NR_DIR}/cl-{src_num}.nr"
                dest_nr_path = os.path.join(
                    "/root/.eth2/imported_records/", f"cl-{src_num}.nr"
                )
                copy_file_to_container(dst_name, src_nr_path, dest_nr_path)

    print("\nSetup complete. Loggers are still running.")
    try:
        while True:
            time.sleep(1)
    except KeyboardInterrupt:
        print("Shutting down...")


if __name__ == "__main__":
    main()
