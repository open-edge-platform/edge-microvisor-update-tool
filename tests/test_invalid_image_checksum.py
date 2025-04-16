import os
import unittest
import subprocess
import gzip
import hashlib

# Path to the script to be tested
SCRIPT_PATH = "./os-update-tool.sh"

class TestABUpdateTool(unittest.TestCase):
    image_path = "valid_image.raw.gz"

    @classmethod
    def setUpClass(cls):
        """
        Set up the environment for testing, including creating the empty raw.gz file.
        """
        # Create an empty .raw.gz file
        cls.create_empty_raw_gz(cls.image_path)

    @classmethod
    def tearDownClass(cls):
        """
        Clean up the environment after testing, including removing test files.
        """
        # Remove the test image file
        if os.path.exists(cls.image_path):
            os.remove(cls.image_path)

    @staticmethod
    def create_empty_raw_gz(file_path):
        """
        Create an empty .raw.gz file for testing purposes.

        Args:
            file_path (str): Path to the .raw.gz file to create.
        """
        with gzip.open(file_path, "wb") as gz_file:
            pass  # Write nothing to create an empty file

    @staticmethod
    def calculate_sha256(file_path):
        """
        Calculate the SHA-256 checksum of a file.

        Args:
            file_path (str): Path to the file to calculate the checksum for.

        Returns:
            str: The SHA-256 checksum of the file.
        """
        sha256_hash = hashlib.sha256()
        with open(file_path, "rb") as f:
            for byte_block in iter(lambda: f.read(4096), b""):
                sha256_hash.update(byte_block)
        return sha256_hash.hexdigest()

    def run_ab_update_tool(self, image_path, checksum_value):
        """
        Run the AB update tool with the specified image path and checksum value, and capture its output.
        """
        try:
            result = subprocess.run(
                ["sudo", SCRIPT_PATH, "-w", "-u", image_path, "-s", checksum_value],
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                text=True,
                check=True,
            )
            return result.returncode, result.stdout, result.stderr
        except subprocess.CalledProcessError as e:
            return e.returncode, e.stdout, e.stderr

    def test_invalid_checksum_handling(self):
        """
        Test: Handle invalid checksum for the image.
        """
        print("Running: test_invalid_checksum_handling")

        # Calculate the valid checksum of the image file
        valid_checksum = self.calculate_sha256(self.image_path)

        # Create an invalid checksum by altering the valid one
        invalid_checksum = valid_checksum[:-1] + ('0' if valid_checksum[-1] != '0' else '1')

        # Run the AB update tool with the invalid checksum
        returncode, stdout, stderr = self.run_ab_update_tool(self.image_path, invalid_checksum)

        # Assert: Expect a non-zero exit code due to checksum mismatch
        self.assertNotEqual(returncode, 0, "Invalid checksum was not handled correctly.")
        print("PASS: Detected and handled invalid checksum. Err:", stderr)


if __name__ == "__main__":
    unittest.main()