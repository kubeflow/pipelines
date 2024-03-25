from typing import NamedTuple
from kfp.components import InputPath, OutputPath, create_component_from_func


def split_rows_into_subsets(
    table_path: InputPath("CSV"),
    split_1_path: OutputPath("CSV"),
    split_2_path: OutputPath("CSV"),
    split_3_path: OutputPath("CSV"),
    fraction_1: float,
    fraction_2: float = None,
    random_seed: int = 0,
) -> NamedTuple(
    "Outputs",
    [
        ("split_1_count", int),
        ("split_2_count", int),
        ("split_3_count", int),
    ],
):
    """Splits the data table according to the split fractions.
    Args:
        fraction_1: The proportion of the lines to put into the 1st split. Range: [0, 1]
        fraction_2: The proportion of the lines to put into the 2nd split. Range: [0, 1]
            If fraction_2 is not specified, then fraction_2 = 1 - fraction_1.
            The remaining lines go to the 3rd split (if any).
    """
    import random

    random.seed(random_seed)

    SHUFFLE_BUFFER_SIZE = 10000

    num_splits = 3

    if fraction_1 < 0 or fraction_1 > 1:
        raise ValueError("fraction_1 must be in between 0 and 1.")

    if fraction_2 is None:
        fraction_2 = 1 - fraction_1
    if fraction_2 < 0 or fraction_2 > 1:
        raise ValueError("fraction_2 must be in between 0 and 1.")

    fraction_3 = 1 - fraction_1 - fraction_2

    fractions = [
        fraction_1,
        fraction_2,
        fraction_3,
    ]

    assert sum(fractions) == 1

    written_line_counts = [0] * num_splits

    output_files = [
        open(split_1_path, "wb"),
        open(split_2_path, "wb"),
        open(split_3_path, "wb"),
    ]

    with open(table_path, "rb") as input_file:
        # Writing the headers
        header_line = input_file.readline()
        for output_file in output_files:
            output_file.write(header_line)

        while True:
            line_buffer = []
            for i in range(SHUFFLE_BUFFER_SIZE):
                line = input_file.readline()
                if not line:
                    break
                line_buffer.append(line)

            # We need to exactly partition the lines between the output files
            # To overcome possible systematic bias, we could calculate the total numbers
            # of lines written to each file and take that into account.
            num_read_lines = len(line_buffer)
            number_of_lines_for_files = [0] * num_splits
            # List that will have the index of the destination file for each line
            file_index_for_line = []
            remaining_lines = num_read_lines
            remaining_fraction = 1
            for i in range(num_splits):
                number_of_lines_for_file = (
                    round(remaining_lines * (fractions[i] / remaining_fraction))
                    if remaining_fraction > 0
                    else 0
                )
                number_of_lines_for_files[i] = number_of_lines_for_file
                remaining_lines -= number_of_lines_for_file
                remaining_fraction -= fractions[i]
                file_index_for_line.extend([i] * number_of_lines_for_file)

            assert remaining_lines == 0, f"{remaining_lines}"
            assert len(file_index_for_line) == num_read_lines

            random.shuffle(file_index_for_line)

            for i in range(num_read_lines):
                output_files[file_index_for_line[i]].write(line_buffer[i])
                written_line_counts[file_index_for_line[i]] += 1

            # Exit if the file ended before we were able to fully fill the buffer
            if len(line_buffer) != SHUFFLE_BUFFER_SIZE:
                break

    for output_file in output_files:
        output_file.close()

    return written_line_counts


if __name__ == "__main__":
    import os
    import re
    # Fixing google3 paths
    if "BUILD_WORKSPACE_DIRECTORY" in os.environ:
        rel_path = re.sub(".*/(?=third_party/)", "", os.path.dirname(__file__))
        os.chdir(os.path.join(os.environ["BUILD_WORKSPACE_DIRECTORY"], rel_path))

    split_rows_into_subsets_op = create_component_from_func(
        split_rows_into_subsets,
        base_image="python:3.9",
        packages_to_install=[],
        output_component_file="component.yaml",
    )
