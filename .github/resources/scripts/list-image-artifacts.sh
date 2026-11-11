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

# Revalidate every page without starting a separate JSON parser. gh returns
# nonzero for HTTP 304; accept that only for an exact-URL cached response.
# The caller owns cache_dir for this wait invocation. Print no partial inventory.
list_image_artifacts() {
  local cache_dir=$1 page=1 url response line status etag next body
  local cached_url cached_etag in_headers request_ok inventory=""
  local retry_after remaining reset now delay
  rm -f "$cache_dir/retry-after" || return 1
  local -a command
  while true; do
    url="repos/${GITHUB_REPOSITORY}/actions/runs/${GITHUB_RUN_ID}/artifacts?per_page=100&page=${page}"
    cached_url="" cached_etag=""
    if [[ -f "$cache_dir/$page.url" && -f "$cache_dir/$page.etag" &&
          -f "$cache_dir/$page.body" && -f "$cache_dir/$page.next" ]]; then
      cached_url=$(<"$cache_dir/$page.url") || return 1
      cached_etag=$(<"$cache_dir/$page.etag") || return 1
    fi
    command=(gh api --include "$url" --jq
      '.artifacts | if type != "array" then error("Invalid artifact inventory") else .[] | if (.name | type) == "string" then .name else error("Invalid artifact name") end end')
    if [[ "$cached_url" == "$url" && -n "$cached_etag" ]]; then
      command+=(-H "If-None-Match: $cached_etag")
    fi
    request_ok=true
    # Preserve the header/body separator even for an empty LF-only 200 body.
    response=$("${command[@]}" 2> "$cache_dir/request-error" && printf '.') || request_ok=false
    [[ "$request_ok" != true ]] || response=${response%.}
    response=${response//$'\r'/}
    status="" etag="" next=false body="" in_headers=true
    retry_after="" remaining="" reset=""
    while IFS= read -r line; do
      if [[ -z "$status" ]]; then
        if [[ "$line" =~ ^HTTP/[0-9.]+[[:space:]]([0-9]{3}) ]]; then
          status=${BASH_REMATCH[1]}
        else
          echo 'Artifact API returned no HTTP status' >&2
          return 1
        fi
      elif [[ "$in_headers" == true ]]; then
        case "$line" in
          '') in_headers=false ;;
          [Ee][Tt][Aa][Gg]:*) etag=${line#*: } ;;
          [Rr][Ee][Tt][Rr][Yy]-[Aa][Ff][Tt][Ee][Rr]:*) retry_after=${line#*: } ;;
          [Xx]-[Rr][Aa][Tt][Ee][Ll][Ii][Mm][Ii][Tt]-[Rr][Ee][Mm][Aa][Ii][Nn][Ii][Nn][Gg]:*) remaining=${line#*: } ;;
          [Xx]-[Rr][Aa][Tt][Ee][Ll][Ii][Mm][Ii][Tt]-[Rr][Ee][Ss][Ee][Tt]:*) reset=${line#*: } ;;
          [Ll][Ii][Nn][Kk]:*)
            [[ "$line" != *'rel="next"'* ]] || next=true
            ;;
        esac
      else
        [[ -z "$line" ]] || body+="$line"$'\n'
      fi
    done <<< "$response"
    if [[ "$status" == 304 && "$cached_url" == "$url" && -n "$cached_etag" ]]; then
      body=$(<"$cache_dir/$page.body") || return 1
      [[ -z "$body" ]] || body+=$'\n'
      next=$(<"$cache_dir/$page.next") || return 1
    elif [[ "$status" == 200 && "$request_ok" == true && "$in_headers" == false ]]; then
      # Invalidate first and publish the validator last. A failed cache write
      # must never pair a new ETag with an older inventory. Explicit checks are
      # required because the caller invokes this function in a conditional.
      : > "$cache_dir/$page.etag" || return 1
      printf '%s' "$url" > "$cache_dir/$page.url" || return 1
      printf '%s' "$body" > "$cache_dir/$page.body" || return 1
      printf '%s' "$next" > "$cache_dir/$page.next" || return 1
      printf '%s' "$etag" > "$cache_dir/$page.etag" || return 1
    else
      echo "Artifact API request failed (HTTP ${status:-unknown})" >&2
      if [[ "$status" == 403 || "$status" == 429 ]]; then
        # All responses may contain reset headers; only an exhausted primary
        # limit makes that reset relevant. Secondary limits use Retry-After.
        if [[ "$status" == 429 || "$remaining" == 0 || -n "$retry_after" ]] ||
            grep -qi 'rate limit' "$cache_dir/request-error"; then
          delay=0
          if [[ "$retry_after" =~ ^[0-9]{1,10}$ ]]; then
            delay=$((10#$retry_after))
          fi
          if [[ "$remaining" == 0 && "$reset" =~ ^[0-9]{1,10}$ ]]; then
            now=$(date +%s) || return 1
            reset=$((10#$reset - now))
            if (( reset > delay )); then delay=$reset; fi
          fi
          printf '%s' "$delay" > "$cache_dir/retry-after" || return 1
        else
          echo "Artifact access was denied; verify the workflow actions:read permission." >&2
          return 2
        fi
      elif [[ "$status" == 401 || "$status" == 404 ]]; then
        echo "Artifact inventory is inaccessible; verify workflow permissions and run ID." >&2
        return 2
      fi
      return 1
    fi
    inventory+="$body"
    [[ "$next" == true ]] || break
    page=$((page + 1))
  done
  printf '%s' "$inventory"
}
