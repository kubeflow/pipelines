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
import * as fsp from 'fs/promises';
import * as path from 'path';
import * as crypto from 'crypto';

/**
 * PodEventsCache provides persistence for pod events, status, and state history.
 *
 * ARCHITECTURE (V5 rewrite):
 * - ALL file I/O is async (fs/promises) to never block the event loop
 * - In-memory index maps runId → pod entries, eliminating full directory scans
 * - Atomic writes (write to temp file, then rename) prevent JSON corruption
 * - Per-file write locks prevent concurrent write races
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
const ERROR_REASONS = new Set([
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
]);

export function isErrorReason(reason: string): boolean {
  return ERROR_REASONS.has(reason);
}

// ─── In-Memory Index ────────────────────────────────────────────────
// Maps "namespace/runId" → Map<podName, IndexEntry>
// This eliminates the need to scan every file for runId lookups.

interface IndexEntry {
  podName: string;
  namespace: string;
  runId: string;
  taskName?: string;
  lastUpdated: number;
  fileName: string;
}

// runKey ("namespace/runId") → Map<podName, IndexEntry>
const runIndex = new Map<string, Map<string, IndexEntry>>();
// podKey ("namespace/podName") → fileName
const podIndex = new Map<string, string>();

let indexInitialized = false;
let indexInitPromise: Promise<void> | null = null;

function runKey(namespace: string, runId: string): string {
  return `${namespace}/${runId}`;
}

function podKey(namespace: string, podName: string): string {
  return `${namespace}/${podName}`;
}

function addToIndex(entry: IndexEntry): void {
  // Add to runIndex
  if (entry.runId) {
    const key = runKey(entry.namespace, entry.runId);
    let podMap = runIndex.get(key);
    if (!podMap) {
      podMap = new Map();
      runIndex.set(key, podMap);
    }
    podMap.set(entry.podName, entry);
  }
  // Add to podIndex
  podIndex.set(podKey(entry.namespace, entry.podName), entry.fileName);
}

function removeFromIndex(namespace: string, podName: string, runId?: string): void {
  podIndex.delete(podKey(namespace, podName));
  if (runId) {
    const key = runKey(namespace, runId);
    const podMap = runIndex.get(key);
    if (podMap) {
      podMap.delete(podName);
      if (podMap.size === 0) {
        runIndex.delete(key);
      }
    }
  }
}

// ─── Per-File Write Locks ───────────────────────────────────────────
// Prevents concurrent writes to the same cache file.
const writeLocks = new Map<string, Promise<void>>();

async function withWriteLock(filePath: string, fn: () => Promise<void>): Promise<void> {
  // Chain onto existing lock for this file
  const existing = writeLocks.get(filePath) || Promise.resolve();
  const next = existing.then(fn, fn); // Run fn regardless of previous result
  writeLocks.set(filePath, next);
  try {
    await next;
  } finally {
    // Clean up if this is still the latest lock
    if (writeLocks.get(filePath) === next) {
      writeLocks.delete(filePath);
    }
  }
}

// ─── Helpers ────────────────────────────────────────────────────────

// Ensure cache directory exists (sync, only at startup)
function ensureCacheDirSync(): void {
  try {
    if (!fs.existsSync(CACHE_DIR)) {
      fs.mkdirSync(CACHE_DIR, { recursive: true });
    }
  } catch (err) {
    console.error('[PodEventsCache] Failed to create cache directory:', err);
  }
}

async function ensureCacheDir(): Promise<void> {
  try {
    await fsp.mkdir(CACHE_DIR, { recursive: true });
  } catch (err) {
    // EEXIST is fine
    if ((err as NodeJS.ErrnoException).code !== 'EEXIST') {
      console.error('[PodEventsCache] Failed to create cache directory:', err);
    }
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
 * Atomic write: write to temp file then rename.
 * Prevents corrupted JSON from partial writes.
 */
async function atomicWriteFile(filePath: string, data: string): Promise<void> {
  const tmpPath = filePath + '.' + crypto.randomBytes(4).toString('hex') + '.tmp';
  await fsp.writeFile(tmpPath, data, 'utf8');
  await fsp.rename(tmpPath, filePath);
}

/**
 * Safely read and parse a cache file. Returns null on any error.
 */
async function readCacheFile(filePath: string): Promise<CachedPodInfo | null> {
  try {
    const content = await fsp.readFile(filePath, 'utf8');
    if (!content || content.trim().length === 0) {
      // Delete empty file
      await fsp.unlink(filePath).catch(() => {});
      return null;
    }
    const parsed = JSON.parse(content) as CachedPodInfo;
    if (!parsed.stateHistory) {
      parsed.stateHistory = [];
    }
    return parsed;
  } catch (err) {
    if ((err as NodeJS.ErrnoException).code === 'ENOENT') {
      return null;
    }
    // Corrupted file - delete it
    await fsp.unlink(filePath).catch(() => {});
    return null;
  }
}

// ─── Index Initialization ───────────────────────────────────────────

/**
 * Build the in-memory index by scanning existing cache files once at startup.
 * This is the only time we do a full directory scan.
 */
export async function initIndex(): Promise<void> {
  if (indexInitialized) return;
  if (indexInitPromise) return indexInitPromise;

  indexInitPromise = (async () => {
    await ensureCacheDir();
    const startTime = Date.now();

    try {
      const files = await fsp.readdir(CACHE_DIR);
      const jsonFiles = files.filter(f => f.endsWith('.json'));

      let indexed = 0;
      let corrupted = 0;

      // Process files in batches to avoid overwhelming the event loop
      const BATCH_SIZE = 50;
      for (let i = 0; i < jsonFiles.length; i += BATCH_SIZE) {
        const batch = jsonFiles.slice(i, i + BATCH_SIZE);
        const results = await Promise.all(
          batch.map(async (file) => {
            const filePath = path.join(CACHE_DIR, file);
            const cached = await readCacheFile(filePath);
            if (!cached) {
              return { ok: false };
            }

            // Skip expired
            if (Date.now() - cached.lastUpdated > CACHE_TTL_MS) {
              await fsp.unlink(filePath).catch(() => {});
              return { ok: false };
            }

            return { ok: true, cached, file };
          }),
        );

        for (const result of results) {
          if (result.ok && result.cached && result.file) {
            addToIndex({
              podName: result.cached.podName,
              namespace: result.cached.namespace,
              runId: result.cached.runId || '',
              taskName: result.cached.taskName,
              lastUpdated: result.cached.lastUpdated,
              fileName: result.file,
            });
            indexed++;
          } else {
            corrupted++;
          }
        }

        // Yield to the event loop between batches
        if (i + BATCH_SIZE < jsonFiles.length) {
          await new Promise(resolve => setImmediate(resolve));
        }
      }

      const elapsed = Date.now() - startTime;
      console.log(
        `[PodEventsCache] Index initialized: ${indexed} entries indexed, ${corrupted} skipped/corrupted (${elapsed}ms)`,
      );
    } catch (err) {
      console.error('[PodEventsCache] Failed to initialize index:', err);
    }

    indexInitialized = true;
    indexInitPromise = null;
  })();

  return indexInitPromise;
}

// ─── Public API (all async) ─────────────────────────────────────────

/**
 * Save pod status and events to cache.
 * Tracks state history to preserve the full pod lifecycle.
 * Uses atomic writes and per-file locking.
 */
export async function savePodInfo(
  podName: string,
  namespace: string,
  status: CachedPodStatus | null,
  events: CachedPodEvent[],
  runId?: string,
  taskName?: string,
): Promise<void> {
  await initIndex();

  const cachePath = getCachePath(namespace, podName);
  const now = Date.now();

  await withWriteLock(cachePath, async () => {
    try {
      // Read existing cache (async)
      const existingCache = await readCacheFile(cachePath);

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

      // Merge events: keep all unique events
      let mergedEvents = events;
      if (existingCache && existingCache.events && existingCache.events.length > 0) {
        const newEventKeys = new Set(
          events.map(e => `${e.type}:${e.reason}:${e.message.substring(0, 100)}`),
        );
        const preservedEvents = existingCache.events.filter(
          e => !newEventKeys.has(`${e.type}:${e.reason}:${e.message.substring(0, 100)}`),
        );
        mergedEvents = [...events, ...preservedEvents];
      }

      const resolvedRunId = runId || existingCache?.runId;
      const resolvedTaskName = taskName || existingCache?.taskName;

      const cachedInfo: CachedPodInfo = {
        podName,
        namespace,
        runId: resolvedRunId,
        taskName: resolvedTaskName,
        status,
        events: mergedEvents,
        stateHistory,
        lastUpdated: now,
      };

      // Atomic write
      await atomicWriteFile(cachePath, JSON.stringify(cachedInfo));

      // Update in-memory index
      const fileName = `${getCacheKey(namespace, podName)}.json`;
      addToIndex({
        podName,
        namespace,
        runId: resolvedRunId || '',
        taskName: resolvedTaskName,
        lastUpdated: now,
        fileName,
      });
    } catch (err) {
      console.error('[PodEventsCache] Failed to save pod info:', err);
    }
  });
}

/**
 * Get cached pod info by pod name (async, single file read).
 */
export async function getCachedPodInfo(
  namespace: string,
  podName: string,
): Promise<CachedPodInfo | null> {
  await initIndex();

  const cachePath = getCachePath(namespace, podName);

  try {
    const cached = await readCacheFile(cachePath);
    if (!cached) return null;

    // Check if cache has expired
    if (Date.now() - cached.lastUpdated > CACHE_TTL_MS) {
      await fsp.unlink(cachePath).catch(() => {});
      removeFromIndex(namespace, podName, cached.runId);
      return null;
    }

    return cached;
  } catch (err) {
    console.error('[PodEventsCache] Failed to read cached pod info:', err);
    return null;
  }
}

/**
 * Get cached pod info by run ID (async, uses in-memory index).
 * Instead of scanning ALL files, we look up the index first.
 */
export async function getCachedPodInfoByRunId(
  namespace: string,
  runId: string,
  taskName?: string,
): Promise<CachedPodInfo | null> {
  await initIndex();

  const key = runKey(namespace, runId);
  const podMap = runIndex.get(key);

  if (!podMap || podMap.size === 0) {
    return null;
  }

  let bestMatch: CachedPodInfo | null = null;
  let bestScore = -1;

  // Only read files for pods in this run (from the index)
  for (const [, entry] of podMap) {
    // Quick filter by taskName from index metadata
    if (taskName && entry.taskName !== taskName) continue;

    // Check expiry from index metadata
    if (Date.now() - entry.lastUpdated > CACHE_TTL_MS) continue;

    const filePath = path.join(CACHE_DIR, entry.fileName);
    const cached = await readCacheFile(filePath);
    if (!cached) {
      // File gone or corrupted, remove from index
      removeFromIndex(entry.namespace, entry.podName, runId);
      continue;
    }

    if (Date.now() - cached.lastUpdated > CACHE_TTL_MS) {
      await fsp.unlink(filePath).catch(() => {});
      removeFromIndex(entry.namespace, entry.podName, runId);
      continue;
    }

    let score = 0;
    if (taskName && cached.taskName === taskName) score += 100;
    if (cached.status?.reason && isErrorReason(cached.status.reason)) score += 50;
    if (cached.status?.phase && cached.status.phase !== 'Succeeded') score += 25;
    if (cached.stateHistory?.some(s => s.isError)) score += 30;
    score += Math.max(0, 10 - Math.floor((Date.now() - cached.lastUpdated) / (60 * 60 * 1000)));

    if (score > bestScore) {
      bestScore = score;
      bestMatch = cached;
    }
  }

  if (bestMatch) {
    console.log(`[PodEventsCache] Found cached pod for runId=${runId}, taskName=${taskName}: ${bestMatch.podName}`);
  }

  return bestMatch;
}

/**
 * Get cached entries matching a given run ID and task name (async, index-based).
 */
async function getCachedEntriesForRunTask(
  namespace: string,
  runId: string,
  taskName?: string,
): Promise<CachedPodInfo[]> {
  await initIndex();

  const key = runKey(namespace, runId);
  const podMap = runIndex.get(key);

  if (!podMap || podMap.size === 0) {
    return [];
  }

  const results: CachedPodInfo[] = [];

  for (const [, entry] of podMap) {
    if (taskName && entry.taskName !== taskName) continue;
    if (Date.now() - entry.lastUpdated > CACHE_TTL_MS) continue;

    const filePath = path.join(CACHE_DIR, entry.fileName);
    const cached = await readCacheFile(filePath);
    if (!cached) {
      removeFromIndex(entry.namespace, entry.podName, runId);
      continue;
    }

    if (Date.now() - cached.lastUpdated > CACHE_TTL_MS) {
      await fsp.unlink(filePath).catch(() => {});
      removeFromIndex(entry.namespace, entry.podName, runId);
      continue;
    }

    results.push(cached);
  }

  return results;
}

/**
 * Get merged state history from all cached pods for a given run ID and task name (async).
 */
export async function getMergedStateHistory(
  namespace: string,
  runId: string,
  taskName?: string,
): Promise<PodStateSnapshot[]> {
  try {
    const entries = await getCachedEntriesForRunTask(namespace, runId, taskName);
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
 * Get merged events from all cached pods for a given run ID and task name (async).
 */
export async function getMergedEvents(
  namespace: string,
  runId: string,
  taskName?: string,
): Promise<CachedPodEvent[]> {
  try {
    const entries = await getCachedEntriesForRunTask(namespace, runId, taskName);
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

/**
 * Clean up expired cache files (async).
 */
export async function cleanupExpiredCache(): Promise<void> {
  await ensureCacheDir();

  try {
    const files = await fsp.readdir(CACHE_DIR);
    let cleaned = 0;

    // Process in batches
    const BATCH_SIZE = 50;
    for (let i = 0; i < files.length; i += BATCH_SIZE) {
      const batch = files.slice(i, i + BATCH_SIZE);
      await Promise.all(
        batch.map(async (file) => {
          if (!file.endsWith('.json')) return;

          const filePath = path.join(CACHE_DIR, file);
          const cached = await readCacheFile(filePath);
          if (!cached) return;

          if (Date.now() - cached.lastUpdated > CACHE_TTL_MS) {
            await fsp.unlink(filePath).catch(() => {});
            removeFromIndex(cached.namespace, cached.podName, cached.runId);
            cleaned++;
          }
        }),
      );

      // Yield between batches
      if (i + BATCH_SIZE < files.length) {
        await new Promise(resolve => setImmediate(resolve));
      }
    }

    if (cleaned > 0) {
      console.log(`[PodEventsCache] Cleaned up ${cleaned} expired cache files`);
    }
  } catch (err) {
    console.error('[PodEventsCache] Failed to cleanup expired cache:', err);
  }
}

// Run cleanup every hour
setInterval(() => {
  cleanupExpiredCache().catch(err => {
    console.error('[PodEventsCache] Cleanup error:', err);
  });
}, 60 * 60 * 1000);
