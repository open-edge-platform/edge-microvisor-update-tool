import os
import unittest
import subprocess

# Path to the script to be tested
SCRIPT_PATH = "./os-update-tool.sh"

class TestABUpdateTool(unittest.TestCase):
    custom_bin = "/opt/bin"

    @classmethod
    def setUpClass(cls):
        """
        Set up the environment for testing, including creating the empty raw.gz file and custom failing_gzip script.
        """
        # Ensure /opt/bin directory exists
        os.makedirs(cls.custom_bin, exist_ok=True)

        # Modify the PATH to prioritize /opt/bin
        os.environ["PATH"] = f"{cls.custom_bin}:{os.environ['PATH']}"

    @classmethod
    def run_ab_update_tool_test_versioncheck(self):
        """
        Run the AB update tool with the specified image path and capture its output.
        """
        try:
            result = subprocess.run(
                ["sudo", SCRIPT_PATH, "-h"],
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                text=True,
                check=True,
            )
            return result.returncode, result.stdout, result.stderr
        except subprocess.CalledProcessError as e:
            return e.returncode, e.stdout, e.stderr

    def test_versioncheck_error_handling(self):
        """
        Test: Handle versioncheck operation
        """
        print("Running: test_versioncheck")

        # Run the AB update tool
        returncode, stdout, stderr = self.run_ab_update_tool_test_versioncheck()
        # Print stdout and stderr for debugging
        print("STDOUT:", stdout)
        print("STDERR:", stderr)
        first_line = stdout.splitlines()[0] if stdout else ""
        self.assertIn("os-update-tool ver", first_line, "First line of stdout does not contain version information. ERR:" + first_line)
        print("PASS: Detected Version Information")

if __name__ == "__main__":
    unittest.main()
