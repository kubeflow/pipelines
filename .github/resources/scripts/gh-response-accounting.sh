#!/usr/bin/env bash
# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#     http://www.apache.org/licenses/LICENSE-2.0
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Record only fixed categories, numeric status/quota fields and process exits.
# The journal survives command substitutions; response bodies and URLs never enter it.
record_gh_response() {
  local journal=$1 category=$2 exit_code=$3 response=$4 mode=${5:-headers}
  local line response_status status="" in_headers=false limit="" remaining="" used="" reset="" resource=""
  [[ "$category" == artifacts || "$category" == jobs ]] || return 0
  [[ "$exit_code" =~ ^[0-9]{1,3}$ ]] || exit_code=255
  printf 'I\t%s\t%s\n' "$category" "$exit_code" >> "$journal" || true
  response=${response//$'\r'/}
  while IFS= read -r line; do
    if [[ "$line" =~ ^HTTP/[0-9.]+[[:space:]]([0-9]{3})([[:space:]]|$) ]]; then
      response_status=${BASH_REMATCH[1]}
      if [[ -n "$status" ]]; then
        write_gh_response_record "$journal" "$category" "$status" "$limit" "$remaining" "$used" "$reset" "$resource"
      fi
      status=$response_status
      in_headers=true
      limit="" remaining="" used="" reset="" resource=""
    elif [[ "$in_headers" == true ]]; then
      case "$line" in
        '') in_headers=false ;;
        [Xx]-[Rr][Aa][Tt][Ee][Ll][Ii][Mm][Ii][Tt]-[Ll][Ii][Mm][Ii][Tt]:*) limit=${line#*: } ;;
        [Xx]-[Rr][Aa][Tt][Ee][Ll][Ii][Mm][Ii][Tt]-[Rr][Ee][Mm][Aa][Ii][Nn][Ii][Nn][Gg]:*) remaining=${line#*: } ;;
        [Xx]-[Rr][Aa][Tt][Ee][Ll][Ii][Mm][Ii][Tt]-[Uu][Ss][Ee][Dd]:*) used=${line#*: } ;;
        [Xx]-[Rr][Aa][Tt][Ee][Ll][Ii][Mm][Ii][Tt]-[Rr][Ee][Ss][Ee][Tt]:*) reset=${line#*: } ;;
        [Xx]-[Rr][Aa][Tt][Ee][Ll][Ii][Mm][Ii][Tt]-[Rr][Ee][Ss][Oo][Uu][Rr][Cc][Ee]:*) resource=${line#*: } ;;
      esac
    elif [[ "$mode" == body ]]; then
      printf '%s\n' "$line"
    fi
  done <<< "$response"
  if [[ -n "$status" ]]; then
    write_gh_response_record "$journal" "$category" "$status" "$limit" "$remaining" "$used" "$reset" "$resource"
  fi
  return 0
}

write_gh_response_record() {
  local journal=$1 category=$2 status=$3 resource=$8 value
  shift 3
  local -a quota=()
  for value in "$1" "$2" "$3" "$4"; do
    if [[ "$value" =~ ^[0-9]{1,10}$ ]]; then
      quota+=("$((10#$value))")
    else
      quota+=("-")
    fi
  done
  case "$resource" in core|search|graphql|code_search) ;; *) resource=unknown ;; esac
  printf 'R\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n' \
    "$category" "$status" "${quota[@]}" "$resource" >> "$journal" || true
  return 0
}
