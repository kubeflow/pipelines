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

/**
 * Background Pod Watcher (V5 - Non-blocking)
 *
 * Automatically caches pod status/events for RUNNING KFP pipeline pods.
 * Uses the shared K8s client and fully async cache operations.
 *
 * Key improvements over V4:
 * - Uses shared K8s client from k8s-helper (no duplicate connections)
 * - All cache operations are async (never blocks event loop)
 * - Bounded podStates map with timestamp-based eviction
 * - Concurrency-limited pod processing
 */

import { V1Pod } from '@kubernetes/client-node';
import * as fs from 'fs';
import { TEST_ONLY } from './k8s-helper';
import * as podEventsCache from './pod-events-cache';

// Use the shared K8s client from k8s-helper
const k8sV1Client = TEST_ONLY.k8sV1Client;

// Configuration from environment variables
const POD_WATCHER_ENABLED = process.env.POD_WATCHER_ENABLED !== 'false';
const POD_WATCHER_INTERVAL_MS = parseInt(process.env.POD_WATCHER_INTERVAL_MS || '30000', 10);
const POD_WATCHER_NAMESPACES = (process.env.POD_WATCHER_NAMESPACES || '').split(',').filter(Boolean);

// If no namespaces configured, try to detect from service account
const namespaceFilePath = '/var/run/secrets/kubernetes.io/serviceaccount/namespace';
let serverNamespace: string | undefined = undefined;
if (fs.existsSync(namespaceFilePath)) {
  serverNamespace = fs.readFileSync(namespaceFilePath, 'utf-8');
}

// Track pod states to detect changes: podKey -> { phase, resourceVersion, lastSeen }
const podStates = new Map<string, { phase: string; resourceVersion: string; lastSeen: number }>();

// Max entries in podStates map
const MAX_POD_STATES = 500;

// Evict entries older than 1 hour
const POD_STATE_TTL_MS = 60 * 60 * 1000;

/**
 * Extract pod status from K8s pod object
 */
function extractPodStatus(pod: V1Pod): podEventsCache.CachedPodStatus | null {
  if (!pod || !pod.status) return null;

  const parseContainerStatus = (cs: any): podEventsCache.ContainerStatus => ({
    name: cs.name,
    ready: cs.ready,
    restartCount: cs.restartCount,
    state: cs.state ? Object.keys(cs.state)[0] : undefined,
    reason: cs.state?.waiting?.reason || cs.state?.terminated?.reason,
    message: cs.state?.waiting?.message || cs.state?.terminated?.message,
    exitCode: cs.state?.terminated?.exitCode,
  });

  const containerStatuses = (pod.status.containerStatuses || []).map(parseContainerStatus);
  const initContainerStatuses = (pod.status.initContainerStatuses || []).map(parseContainerStatus);

  return {
    phase: pod.status.phase,
    message: pod.status.message,
    reason: pod.status.reason,
    podIP: pod.status.podIP,
    nodeName: pod.spec?.nodeName,
    lastUpdated: Date.now(),
    containerStatuses: [...initContainerStatuses, ...containerStatuses],
  };
}

/**
 * Extract events from K8s event list
 */
function extractEvents(eventList: any): podEventsCache.CachedPodEvent[] {
  if (!eventList || !eventList.items) return [];

  return eventList.items.map((event: any) => ({
    type: event.type,
    reason: event.reason,
    message: event.message,
    firstTimestamp: event.firstTimestamp,
    lastTimestamp: event.lastTimestamp,
    count: event.count || 1,
    source: event.source?.component,
  }));
}

/**
 * Fetch events for a specific pod
 */
async function fetchPodEvents(podName: string, namespace: string): Promise<podEventsCache.CachedPodEvent[]> {
  try {
    const fieldSelector = `involvedObject.name=${podName},involvedObject.kind=Pod`;
    const { body } = await k8sV1Client.listNamespacedEvent(
      namespace,
      undefined,
      undefined,
      undefined,
      fieldSelector,
    );
    return extractEvents(body);
  } catch (err) {
    return [];
  }
}

/**
 * Get runId from pod labels
 */
function getRunIdFromPod(pod: V1Pod): string | undefined {
  const labels = pod.metadata?.labels || {};
  return labels['pipeline/runid'] || labels['pipelines.kubeflow.org/run_id'];
}

/**
 * Get taskName from pod labels/annotations
 */
function getTaskNameFromPod(pod: V1Pod): string | undefined {
  const labels = pod.metadata?.labels || {};
  const annotations = pod.metadata?.annotations || {};
  return labels['pipelines.kubeflow.org/task_name'] || labels['component'] || annotations['pipelines.kubeflow.org/task_path'];
}

/**
 * Check if pod state has changed since last processing
 */
function hasPodChanged(pod: V1Pod): boolean {
  const podName = pod.metadata?.name;
  const namespace = pod.metadata?.namespace;
  if (!podName || !namespace) return false;

  const key = `${namespace}/${podName}`;
  const currentPhase = pod.status?.phase || 'Unknown';
  const currentVersion = pod.metadata?.resourceVersion || '';

  const lastState = podStates.get(key);
  if (!lastState) return true;

  return lastState.phase !== currentPhase || lastState.resourceVersion !== currentVersion;
}

/**
 * Update tracked pod state
 */
function updatePodState(pod: V1Pod): void {
  const podName = pod.metadata?.name;
  const namespace = pod.metadata?.namespace;
  if (!podName || !namespace) return;

  const key = `${namespace}/${podName}`;
  podStates.set(key, {
    phase: pod.status?.phase || 'Unknown',
    resourceVersion: pod.metadata?.resourceVersion || '',
    lastSeen: Date.now(),
  });
}

/**
 * Evict stale entries from podStates map
 */
function evictStalePodStates(): void {
  const now = Date.now();

  // First: remove entries older than TTL
  for (const [key, state] of podStates) {
    if (now - state.lastSeen > POD_STATE_TTL_MS) {
      podStates.delete(key);
    }
  }

  // If still over limit, remove oldest entries
  if (podStates.size > MAX_POD_STATES) {
    const entries = Array.from(podStates.entries())
      .sort((a, b) => a[1].lastSeen - b[1].lastSeen);
    const toRemove = entries.slice(0, podStates.size - MAX_POD_STATES);
    for (const [key] of toRemove) {
      podStates.delete(key);
    }
  }
}

/**
 * Process a single pod: extract status, fetch events, and cache (all async)
 */
async function processPod(pod: V1Pod): Promise<boolean> {
  const podName = pod.metadata?.name;
  const namespace = pod.metadata?.namespace;

  if (!podName || !namespace) return false;

  if (!hasPodChanged(pod)) return false;

  const status = extractPodStatus(pod);
  if (!status) return false;

  const runId = getRunIdFromPod(pod);
  const taskName = getTaskNameFromPod(pod);

  // Fetch existing cache and events in parallel (all async)
  const [existing, events] = await Promise.all([
    podEventsCache.getCachedPodInfo(namespace, podName),
    fetchPodEvents(podName, namespace),
  ]);

  // Save to cache (async)
  await podEventsCache.savePodInfo(
    podName,
    namespace,
    status,
    events.length > 0 ? events : (existing?.events || []),
    runId,
    taskName,
  );

  updatePodState(pod);
  return true;
}

/**
 * List only RUNNING pipeline pods in a namespace
 */
async function listRunningPipelinePods(namespace: string): Promise<V1Pod[]> {
  const allPods: V1Pod[] = [];

  try {
    const { body } = await k8sV1Client.listNamespacedPod(
      namespace,
      undefined,
      undefined,
      undefined,
      'status.phase=Running',
      'pipeline/runid',
      100,
    );

    if (body.items) {
      allPods.push(...body.items);
    }
  } catch (err) {
    // Field selector fallback
    try {
      const { body } = await k8sV1Client.listNamespacedPod(
        namespace,
        undefined,
        undefined,
        undefined,
        undefined,
        'pipeline/runid',
        100,
      );

      if (body.items) {
        for (const pod of body.items) {
          const phase = pod.status?.phase;
          if (phase === 'Running' || phase === 'Pending') {
            allPods.push(pod);
          }
        }
      }
    } catch (fallbackErr) {
      console.error(`[PodWatcher] Error listing pods in ${namespace}:`, fallbackErr);
    }
  }

  return allPods;
}

/**
 * Main watcher loop iteration (fully async, never blocks event loop)
 */
async function watcherLoop(): Promise<void> {
  let namespaces = POD_WATCHER_NAMESPACES;
  if (namespaces.length === 0 && serverNamespace) {
    namespaces = [serverNamespace];
  }

  if (namespaces.length === 0) return;

  let totalProcessed = 0;
  let totalPods = 0;

  for (const namespace of namespaces) {
    try {
      const pods = await listRunningPipelinePods(namespace);
      totalPods += pods.length;

      // Process pods with concurrency limit to avoid overwhelming K8s API
      const CONCURRENCY = 5;
      for (let i = 0; i < pods.length; i += CONCURRENCY) {
        const batch = pods.slice(i, i + CONCURRENCY);
        const results = await Promise.all(
          batch.map(pod => processPod(pod).catch(() => false)),
        );
        totalProcessed += results.filter(Boolean).length;
      }
    } catch (err) {
      // Skip problematic namespaces
    }
  }

  // Evict stale pod state entries
  evictStalePodStates();

  if (totalProcessed > 0) {
    console.log(`[PodWatcher] Cached ${totalProcessed} new/changed pods (${totalPods} running total)`);
  }
}

let watcherIntervalId: NodeJS.Timeout | null = null;

/**
 * Start the background pod watcher
 */
export function startPodWatcher(): void {
  if (!POD_WATCHER_ENABLED) {
    console.log('[PodWatcher] Pod watcher is disabled (POD_WATCHER_ENABLED=false)');
    return;
  }

  if (watcherIntervalId) return;

  const namespaces = POD_WATCHER_NAMESPACES.length > 0 ? POD_WATCHER_NAMESPACES : (serverNamespace ? [serverNamespace] : []);
  console.log(`[PodWatcher] Starting background pod watcher`);
  console.log(`[PodWatcher] - Interval: ${POD_WATCHER_INTERVAL_MS}ms`);
  console.log(`[PodWatcher] - Namespaces: ${namespaces.length > 0 ? namespaces.join(', ') : '(none configured)'}`);
  console.log(`[PodWatcher] - Mode: Running/Pending pods only (async, non-blocking)`);

  // Run immediately on start
  watcherLoop().catch(() => {});

  // Then run on interval
  watcherIntervalId = setInterval(() => {
    watcherLoop().catch(() => {});
  }, POD_WATCHER_INTERVAL_MS);
}

/**
 * Stop the background pod watcher
 */
export function stopPodWatcher(): void {
  if (watcherIntervalId) {
    clearInterval(watcherIntervalId);
    watcherIntervalId = null;
    console.log('[PodWatcher] Background pod watcher stopped');
  }
}
