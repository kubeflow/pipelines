// Copyright 2024 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import * as fs from 'fs';
import * as path from 'path';

/**
 * PodEventsCache provides persistence for pod events, status, and state history.
 *
 * It tracks the full lifecycle of a pod from scheduling to completion,
 * preserving error states (like ImagePullBackOff) even after they resolve.
 * This is critical because K8s events expire after ~1 hour.
 */

// Directory to store cached pod events
const CACHE_DIR = process.env.POD_EVENTS_CACHE_DIR || '/tmp/kfp-pod-events-cache';

// How long to keep cached events (30 days in milliseconds)
const CACHE_TTL_MS = 30 * 24 * 60 * 60 * 1000;

export interface ContainerStatus {
  name: string;
  ready: boolean;
  restartCount: number;
  state?: string;
  reason?: string;
  message?: string;
  exitCode?: number;
}

export interface CachedPodStatus {
  phase?: string;
  message?: string;
  reason?: string;
  podIP?: string;
  nodeName?: string;
  lastUpdated: number;
  containerStatuses?: ContainerStatus[];
}

export interface CachedPodEvent {
  type: string;
  reason: string;
  message: string;
  firstTimestamp?: string;
  lastTimestamp?: string;
  count: number;
  source?: string;
}

/** A snapshot of the pod's state at a point in time */
export interface PodStateSnapshot {
  timestamp: number;
  phase?: string;
  reason?: string;
  message?: string;
  containerStatuses?: ContainerStatus[];
  isError: boolean;
}

export interface CachedPodInfo {
  podName: string;
  namespace: string;
  runId?: string;
  taskName?: string;
  status: CachedPodStatus | null;
  events: CachedPodEvent[];
  stateHistory: PodStateSnapshot[];
  lastUpdated: number;
}

// Error reasons that indicate a failed/problematic state
const ERROR_REASONS = [
  'ErrImagePull',
  'ImagePullBackOff',
  'CrashLoopBackOff',
  'CreateContainerConfigError',
  'InvalidImageName',
  'CreateContainerError',
  'RunContainerError',
  'PreCreateHookError',
  'PostStartHookError',
  'ContainerCannotRun',
  'OOMKilled',
  'Error',
];

export function isErrorReason(reason: string): boolean {
  return ERROR_REASONS.includes(reason);
}

// Ensure cache directory exists
function ensureCacheDir(): void {
  try {
    if (!fs.existsSync(CACHE_DIR)) {
      fs.mkdirSync(CACHE_DIR, { recursive: true });
    }
  } catch (err) {
    console.error('[PodEventsCache] Failed to create cache directory:', err);
  }
}

// Generate a cache key from pod identifiers
function getCacheKey(namespace: string, podName: string): string {
  return Buffer.from(`${namespace}/${podName}`).toString('base64').replace(/[/+=]/g, '_');
}

// Generate cache file path
function getCachePath(namespace: string, podName: string): string {
  return path.join(CACHE_DIR, `${getCacheKey(namespace, podName)}.json`);
}

/**
 * Create a fingerprint of the current state to detect changes.
 * Includes phase, container states, and error reasons.
 */
function stateFingerprint(status: CachedPodStatus | null): string {
  if (!status) return 'null';
  const containerParts = (status.containerStatuses || [])
    .map(cs => `${cs.name}:${cs.state}:${cs.reason || ''}:${cs.ready}:${cs.exitCode ?? ''}`)
    .sort()
    .join('|');
  return `${status.phase}:${status.reason || ''}:${containerParts}`;
}

/**
 * Check if a status snapshot represents an error state
 */
function isErrorState(status: CachedPodStatus | null): boolean {
  if (!status) return false;
  if (status.reason && isErrorReason(status.reason)) return true;
  if (status.containerStatuses) {
    return status.containerStatuses.some(
      cs => (cs.reason && isErrorReason(cs.reason)) ||
            (cs.state === 'terminated' && cs.exitCode !== undefined && cs.exitCode !== 0),
    );
  }
  return false;
}

/**
 * Save pod status and events to cache.
 * Tracks state history to preserve the full pod lifecycle,
 * including error states that later resolve.
 */
export function savePodInfo(
  podName: string,
  namespace: string,
  status: CachedPodStatus | null,
  events: CachedPodEvent[],
  runId?: string,
  taskName?: string,
): void {
  ensureCacheDir();

  const cachePath = getCachePath(namespace, podName);
  const now = Date.now();

  try {
    // Read existing cache
    let existingCache: CachedPodInfo | null = null;
    if (fs.existsSync(cachePath)) {
      try {
        existingCache = JSON.parse(fs.readFileSync(cachePath, 'utf8'));
      } catch (e) {
        // Ignore parse errors, we'll overwrite
      }
    }

    // Build state history
    let stateHistory: PodStateSnapshot[] = existingCache?.stateHistory || [];

    // Check if state changed since last snapshot
    const lastFingerprint = stateHistory.length > 0
      ? stateFingerprint({
          phase: stateHistory[stateHistory.length - 1].phase,
          reason: stateHistory[stateHistory.length - 1].reason,
          containerStatuses: stateHistory[stateHistory.length - 1].containerStatuses,
          lastUpdated: 0,
        })
      : '';
    const currentFingerprint = stateFingerprint(status);

    if (currentFingerprint !== lastFingerprint && status) {
      stateHistory.push({
        timestamp: now,
        phase: status.phase,
        reason: status.reason,
        message: status.message,
        containerStatuses: status.containerStatuses ? [...status.containerStatuses] : undefined,
        isError: isErrorState(status),
      });
    }

    // Merge events: keep all unique events (Warning events are never removed)
    let mergedEvents = events;
    if (existingCache && existingCache.events && existingCache.events.length > 0) {
      const newEventKeys = new Set(
        events.map(e => `${e.type}:${e.reason}:${e.message.substring(0, 100)}`),
      );

      // Keep events from cache that are not in new events
      const preservedEvents = existingCache.events.filter(
        e => !newEventKeys.has(`${e.type}:${e.reason}:${e.message.substring(0, 100)}`),
      );

      mergedEvents = [...events, ...preservedEvents];
    }

    const cachedInfo: CachedPodInfo = {
      podName,
      namespace,
      runId: runId || existingCache?.runId,
      taskName: taskName || existingCache?.taskName,
      status,
      events: mergedEvents,
      stateHistory,
      lastUpdated: now,
    };

    fs.writeFileSync(cachePath, JSON.stringify(cachedInfo, null, 2));
    console.log(
      `[PodEventsCache] Saved pod info for ${namespace}/${podName}` +
      ` (${mergedEvents.length} events, ${stateHistory.length} state snapshots)`,
    );
  } catch (err) {
    console.error('[PodEventsCache] Failed to save pod info:', err);
  }
}

/**
 * Get cached pod info
 */
export function getCachedPodInfo(
  namespace: string,
  podName: string,
): CachedPodInfo | null {
  const cachePath = getCachePath(namespace, podName);

  try {
    if (!fs.existsSync(cachePath)) {
      return null;
    }

    const cached: CachedPodInfo = JSON.parse(fs.readFileSync(cachePath, 'utf8'));

    // Check if cache has expired
    if (Date.now() - cached.lastUpdated > CACHE_TTL_MS) {
      fs.unlinkSync(cachePath);
      return null;
    }

    // Ensure stateHistory exists for backward compatibility
    if (!cached.stateHistory) {
      cached.stateHistory = [];
    }

    return cached;
  } catch (err) {
    console.error('[PodEventsCache] Failed to read cached pod info:', err);
    return null;
  }
}

/**
 * Get cached pod info by run ID
 */
export function getCachedPodInfoByRunId(
  namespace: string,
  runId: string,
  taskName?: string,
): CachedPodInfo | null {
  ensureCacheDir();

  try {
    const files = fs.readdirSync(CACHE_DIR);
    let bestMatch: CachedPodInfo | null = null;
    let bestScore = -1;

    for (const file of files) {
      if (!file.endsWith('.json')) continue;

      try {
        const cached: CachedPodInfo = JSON.parse(
          fs.readFileSync(path.join(CACHE_DIR, file), 'utf8'),
        );

        if (Date.now() - cached.lastUpdated > CACHE_TTL_MS) continue;
        if (cached.namespace !== namespace) continue;
        if (cached.runId !== runId) continue;

        // When taskName is provided, strictly filter by it to avoid
        // returning a different component's pod
        if (taskName && cached.taskName !== taskName) continue;

        let score = 0;
        if (taskName && cached.taskName === taskName) score += 100;
        if (cached.status?.reason && isErrorReason(cached.status.reason)) score += 50;
        if (cached.status?.phase && cached.status.phase !== 'Succeeded') score += 25;
        // Prefer pods with error history
        if (cached.stateHistory?.some(s => s.isError)) score += 30;
        score += Math.max(0, 10 - Math.floor((Date.now() - cached.lastUpdated) / (60 * 60 * 1000)));

        if (score > bestScore) {
          bestScore = score;
          bestMatch = cached;
        }
      } catch (e) {
        // Skip files that can't be parsed
      }
    }

    return bestMatch;
  } catch (err) {
    console.error('[PodEventsCache] Failed to search cached pod info by run ID:', err);
    return null;
  }
}

/**
 * Clean up expired cache files
 */
export function cleanupExpiredCache(): void {
  ensureCacheDir();

  try {
    const files = fs.readdirSync(CACHE_DIR);
    let cleaned = 0;

    for (const file of files) {
      if (!file.endsWith('.json')) continue;

      try {
        const filePath = path.join(CACHE_DIR, file);
        const cached: CachedPodInfo = JSON.parse(fs.readFileSync(filePath, 'utf8'));

        if (Date.now() - cached.lastUpdated > CACHE_TTL_MS) {
          fs.unlinkSync(filePath);
          cleaned++;
        }
      } catch (e) {
        // Skip files that can't be parsed
      }
    }

    if (cleaned > 0) {
      console.log(`[PodEventsCache] Cleaned up ${cleaned} expired cache files`);
    }
  } catch (err) {
    console.error('[PodEventsCache] Failed to cleanup expired cache:', err);
  }
}

/**
 * Get cached entries matching a given run ID and task name.
 * Strict filtering: when taskName is provided, entries must match exactly.
 */
function getCachedEntriesForRunTask(
  namespace: string,
  runId: string,
  taskName?: string,
): CachedPodInfo[] {
  ensureCacheDir();

  const results: CachedPodInfo[] = [];
  try {
    const files = fs.readdirSync(CACHE_DIR);

    for (const file of files) {
      if (!file.endsWith('.json')) continue;

      try {
        const cached: CachedPodInfo = JSON.parse(
          fs.readFileSync(path.join(CACHE_DIR, file), 'utf8'),
        );

        if (Date.now() - cached.lastUpdated > CACHE_TTL_MS) continue;
        if (cached.namespace !== namespace) continue;
        if (cached.runId !== runId) continue;
        // Strict taskName filter: when provided, cached entry must match
        if (taskName && cached.taskName !== taskName) continue;

        results.push(cached);
      } catch (e) {
        // Skip files that can't be parsed
      }
    }
  } catch (err) {
    console.error('[PodEventsCache] Failed to get cached entries:', err);
  }

  return results;
}

/**
 * Get merged state history from all cached pods for a given run ID and task name.
 * This preserves state transitions from pods that no longer exist (e.g., failed impl pods
 * that were deleted on pipeline termination) by combining history across all pods
 * belonging to the same run/task.
 */
export function getMergedStateHistory(
  namespace: string,
  runId: string,
  taskName?: string,
): PodStateSnapshot[] {
  try {
    const entries = getCachedEntriesForRunTask(namespace, runId, taskName);
    const allSnapshots: PodStateSnapshot[] = [];

    for (const cached of entries) {
      if (cached.stateHistory && cached.stateHistory.length > 0) {
        allSnapshots.push(...cached.stateHistory);
      }
    }

    // Deduplicate by timestamp + phase + reason
    const seen = new Set<string>();
    const unique = allSnapshots.filter(s => {
      const key = `${s.timestamp}:${s.phase}:${s.reason || ''}`;
      if (seen.has(key)) return false;
      seen.add(key);
      return true;
    });

    // Sort by timestamp ascending
    unique.sort((a, b) => a.timestamp - b.timestamp);

    return unique;
  } catch (err) {
    console.error('[PodEventsCache] Failed to get merged state history:', err);
    return [];
  }
}

/**
 * Get merged events from all cached pods for a given run ID and task name.
 * This preserves error events (like ImagePullBackOff details) from pods that
 * were deleted on pipeline termination.
 */
export function getMergedEvents(
  namespace: string,
  runId: string,
  taskName?: string,
): CachedPodEvent[] {
  try {
    const entries = getCachedEntriesForRunTask(namespace, runId, taskName);
    const allEvents: CachedPodEvent[] = [];

    for (const cached of entries) {
      if (cached.events && cached.events.length > 0) {
        allEvents.push(...cached.events);
      }
    }

    // Deduplicate by type + reason + message (first 100 chars)
    const seen = new Set<string>();
    return allEvents.filter(e => {
      const key = `${e.type}:${e.reason}:${e.message.substring(0, 100)}`;
      if (seen.has(key)) return false;
      seen.add(key);
      return true;
    });
  } catch (err) {
    console.error('[PodEventsCache] Failed to get merged events:', err);
    return [];
  }
}

// Run cleanup every hour
setInterval(cleanupExpiredCache, 60 * 60 * 1000);
