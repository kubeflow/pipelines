import sys

from .test_utils.file_utils import FileUtils

print(f"Adding test_data to the sys path")
# Adding test_data to the sys path for it to be available for import in test
# Since *test* is excluded in packaging (in setup.py), this will not part of the published code
sys.path.append(FileUtils.TEST_DATA)
sys.path.append(FileUtils.VALID_PIPELINE_FILES)