import os
import unittest
import subprocess
import gzip

# Path to the script to be tested
SCRIPT_PATH = "./os-update-tool.sh"

class TestABUpdateTool(unittest.TestCase):
    custom_bin = "/opt/bin"
    failing_gzip_path = os.path.join(custom_bin, "gzip")
    image_path = "valid_image.raw.gz"

    @classmethod
    def setUpClass(cls):
        """
        Set up the environment for testing, including creating the empty raw.gz file and custom failing_gzip script.
        """
        # Ensure /opt/bin directory exists
        os.makedirs(cls.custom_bin, exist_ok=True)

        # Create the failing_gzip script
        with open(cls.failing_gzip_path, "w") as f:
            f.write("#!/bin/bash\n")
            f.write('echo "Simulated gzip failure" >&2\n')
            f.write("exit 1\n")

        # Make the script executable
        os.chmod(cls.failing_gzip_path, 0o755)

        # Modify the PATH to prioritize /opt/bin
        os.environ["PATH"] = f"{cls.custom_bin}:{os.environ['PATH']}"

        # Create an empty .raw.gz file
        cls.create_empty_raw_gz(cls.image_path)

    @classmethod
    def tearDownClass(cls):
        """
        Clean up the environment after testing, including removing the custom failing_gzip script and test files.
        """
        if os.path.exists(cls.failing_gzip_path):
            os.remove(cls.failing_gzip_path)
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

    def run_ab_update_tool(self, image_path):
        """
        Run the AB update tool with the specified image path and capture its output.
        """
        try:
            result = subprocess.run(
                ["sudo", SCRIPT_PATH, "-w", "-u", image_path],
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                text=True,
                check=True,
            )
            return result.returncode, result.stdout, result.stderr
        except subprocess.CalledProcessError as e:
            return e.returncode, e.stdout, e.stderr

    def test_gzip_error_handling(self):
        """
        Test: Handle gzip operation errors using the custom failing_gzip.
        """
        print("Running: test_gzip_error_handling")

        # Run the AB update tool
        returncode, stdout, stderr = self.run_ab_update_tool(self.image_path)

        # Assert: Expect a non-zero exit code
        self.assertNotEqual(returncode, 0, "gzip failure was not handled correctly.")
        print("PASS: Detected and handled gzip failure. Err:", stderr)


if __name__ == "__main__":
    unittest.main()