#!/usr/bin/env bash

check_requirements() {
  local req_in="$1"
  local req_txt="$2"

  temp_file=$(mktemp)

  pip-compile - < "$req_in" --output-file="$temp_file" --annotate --resolver=backtracking

  # Compare the generated file with the existing requirements.txt
  if diff -q "$temp_file" "$req_txt" >/dev/null; then
    echo "Success: $req_txt matches the output of pip-compile for $req_in."
    rm "$temp_file"
    exit 0
  else
    echo "Error: $req_txt does not match the output of pip-compile for $req_in."
    echo "Differences:"
    diff "$temp_file" "$req_txt"
    rm "$temp_file"
    exit 1
  fi
}

for req_in_file in $(find . -name "requirements.in"); do

  echo $req_in_file
  req_txt="${req_in_file/requirements.in/requirements.txt}"

  # Check if the corresponding requirements.txt exists
  if [[ -f "$req_txt" ]]; then
    echo "Checking $req_in_file against $req_txt..."
    check_requirements "$req_in_file" "$req_txt"
  else
    echo "Error: $req_txt not found for $req_in_file."
    exit 1
  fi
done

echo "All checks completed successfully."
