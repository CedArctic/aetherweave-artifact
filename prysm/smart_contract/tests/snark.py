import subprocess

SNARKJS = "./node_modules/.bin/snarkjs"


class SnarkError(Exception):
    def __init__(self, message: str) -> None:
        super().__init__(message)


def generate_witness(input_file: str, wasm_file: str, output_dir: str) -> None:
    result = subprocess.run(
        [
            SNARKJS,
            "wtns",
            "calculate",
            wasm_file,
            input_file,
            f"{output_dir}/witness.wtns",
        ],
        capture_output=True,
        text=True,
    )
    if "ERROR" in result.stdout:
        raise SnarkError(f"Witness generation failed: {result.stdout.strip()}")


def generate_proof(
    proving_key: str, witness_file: str, output_dir: str
) -> None:
    result = subprocess.run(
        [
            SNARKJS,
            "groth16",
            "prove",
            proving_key,
            witness_file,
            f"{output_dir}/proof.json",
            f"{output_dir}/public.json",
        ],
        capture_output=True,
        text=True,
    )
    if "ERROR" in result.stdout:
        raise SnarkError(f"Proof generation failed: {result.stdout.strip()}")


def verify_proof(
    verification_key: str, public_file: str, proof_file: str
) -> None:
    result = subprocess.run(
        [
            SNARKJS,
            "groth16",
            "verify",
            verification_key,
            public_file,
            proof_file,
        ],
        capture_output=True,
        text=True,
    )
    if "OK" not in result.stdout:
        raise SnarkError(f"Proof verification failed: {result.stdout.strip()}")
