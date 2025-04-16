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
    def run_ab_update_tool_test_commit(self):
        """
        Run the AB update tool with the specified image path and capture its output.
        """
        try:
            result = subprocess.run(
                ["sudo", SCRIPT_PATH, "-c"],
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                text=True,
                check=True,
            )
            return result.returncode, result.stdout, result.stderr
        except subprocess.CalledProcessError as e:
            return e.returncode, e.stdout, e.stderr

    def test_commit_error_handling(self):
        """
        Test: Handle commit operation errors without proper previous step.
        """
        print("Running: test_commit_error_handling")

        # Run the AB update tool
        returncode, stdout, stderr = self.run_ab_update_tool_test_commit()

        # Assert: Expect a non-zero exit code
        self.assertNotEqual(returncode, 0, "commit failure was not handled correctly.")
        print("PASS: Detected and handled commit failure. FAILURE:", stderr)

if __name__ == "__main__":
    unittest.main()
