import os
import unittest
import subprocess
import threading
import time
import gzip

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
        Clean up the environment after testing, including removing the test files.
        """
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

    def test_parallel_execution(self):
        """
        Test: Attempt to run two instances of the AB update tool in parallel to check for race condition handling.
        """
        print("Running: test_parallel_execution")

        def run_tool_in_thread(results, index):
            returncode, stdout, stderr = self.run_ab_update_tool(self.image_path)
            results[index] = (returncode, stdout, stderr)

        # Dictionary to store results from each thread
        results = {}

        # Create two threads to run the AB update tool in parallel
        thread1 = threading.Thread(target=run_tool_in_thread, args=(results, 1))
        thread2 = threading.Thread(target=run_tool_in_thread, args=(results, 2))

        # Start the first thread
        thread1.start()
        # Start the second thread
        thread2.start()

        # Wait for both threads to complete
        thread1.join()
        thread2.join()

        # Check results from both threads
        for index in [1, 2]:
            returncode, stdout, stderr = results[index]
            if returncode != 0:
                print(f"Thread {index} failed as expected due to concurrent execution. Err: {stdout}{stderr}")
            else:
                print(f"Thread {index} completed successfully. Output: {stdout}")

        # Assert that at least one thread failed due to concurrent execution
        self.assertTrue(
            any(results[index][0] != 0 for index in [1, 2]),
            "Both instances of the script ran successfully, indicating a potential race condition issue."
        )


if __name__ == "__main__":
    unittest.main()