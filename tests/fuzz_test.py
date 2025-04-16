import random
import string
import subprocess
import unittest

# Path to the script to be tested
SCRIPT_PATH = "./os-update-tool.sh"

# Function to generate random strings
def generate_random_string(length):
    return ''.join(random.choices(string.ascii_letters + string.digits + '_-', k=length))

# Function to generate random paths
def generate_random_path(length):
    return f"/tmp/{generate_random_string(length)}"

class TestOsUpdateTool(unittest.TestCase):
    # Number of fuzz test iterations
    NUM_ITERATIONS = 100

    def run_fuzz_test(self, iteration):
        # Generate random path
        random_path = generate_random_path(10)

        # Construct the command
        command = ["sudo", SCRIPT_PATH, "-w", "-u", random_path]

        print(f"Iteration {iteration}: Running: {' '.join(command)}")
        try:
            result = subprocess.run(command, capture_output=True, text=True, check=False)
            print(f"Iteration {iteration}: PASS")
            print(f"Output:\n{result.stdout}")
            return True
        except subprocess.SubprocessError as e:
            print(f"Iteration {iteration}: FAIL")
            print(f"Subprocess Error: {str(e)}")
            return False
        except OSError as e:
            print(f"Iteration {iteration}: FAIL")
            print(f"OS Error: {str(e)}")
            return False
        except ValueError as e:
            print(f"Iteration {iteration}: FAIL")
            print(f"Value Error: {str(e)}")
            return False

    def test_fuzz(self):
        overall_pass = True
        for i in range(1, self.NUM_ITERATIONS + 1):
            if not self.run_fuzz_test(i):
                overall_pass = False
        self.assertTrue(overall_pass, "Overall Test Result: FAIL")

if __name__ == '__main__':
    unittest.main()
    