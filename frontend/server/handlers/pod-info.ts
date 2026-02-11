// Copyright 2020 The Kubeflow Authors
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

import { Handler } from 'express';
import * as k8sHelper from '../k8s-helper';
import * as podEventsCache from '../pod-events-cache';

// Helper to extract pod status from K8s pod object
function extractPodStatus(pod: any): any | null {
  if (!pod || !pod.status) return null;

  const parseContainerStatus = (cs: any) => ({
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

// Helper to extract events from K8s event list
function extractEvents(eventList: any): any[] {
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
 * podInfoHandler retrieves pod info and sends back as JSON format.
 * Also caches the pod status for later retrieval.
 */
export const podInfoHandler: Handler = async (req, res) => {
  const { podname, podnamespace, runid, taskname } = req.query;
  if (!podname) {
    res.status(422).send('podname argument is required');
    return;
  }
  if (!podnamespace) {
    res.status(422).send('podnamespace argument is required');
    return;
  }
  const podName = decodeURIComponent(podname as string);
  const podNamespace = decodeURIComponent(podnamespace as string);
  const runId = runid ? decodeURIComponent(runid as string) : undefined;
  const taskName = taskname ? decodeURIComponent(taskname as string) : undefined;

  const [pod, err] = await k8sHelper.getPod(podName, podNamespace);
  if (err) {
    const { message, additionalInfo } = err;
    console.error(message, additionalInfo);

    // Try to return cached data if K8s pod is not available
    const cached = await podEventsCache.getCachedPodInfo(podNamespace, podName);
    if (cached && cached.status) {
      console.log('[podInfoHandler] Returning cached pod status for', podName);

      const stateHistory = runId
        ? await podEventsCache.getMergedStateHistory(podNamespace, runId, taskName)
        : (cached.stateHistory || []);

      res.status(200).send(
        JSON.stringify({
          metadata: { name: podName, namespace: podNamespace },
          status: {
            phase: cached.status.phase,
            message: cached.status.message,
            reason: cached.status.reason,
            podIP: cached.status.podIP,
            containerStatuses: cached.status.containerStatuses,
          },
          spec: { nodeName: cached.status.nodeName },
          _cached: true,
          _cachedAt: cached.lastUpdated,
          _stateHistory: stateHistory,
        }),
      );
      return;
    }

    res.status(500).send(message);
    return;
  }

  // Cache the pod status (fire-and-forget, don't block response)
  const status = extractPodStatus(pod);
  if (status) {
    podEventsCache.getCachedPodInfo(podNamespace, podName).then(existing => {
      podEventsCache.savePodInfo(
        podName,
        podNamespace,
        status,
        existing?.events || [],
        runId,
        taskName,
      ).catch(() => {});
    }).catch(() => {});
  }

  // Read back state history from cache and attach to response
  const response = JSON.parse(JSON.stringify(pod));
  if (runId) {
    const mergedHistory = await podEventsCache.getMergedStateHistory(podNamespace, runId, taskName);
    if (mergedHistory.length > 0) {
      response._stateHistory = mergedHistory;
    }
  } else {
    const updatedCache = await podEventsCache.getCachedPodInfo(podNamespace, podName);
    if (updatedCache?.stateHistory && updatedCache.stateHistory.length > 0) {
      response._stateHistory = updatedCache.stateHistory;
    }
  }

  res.status(200).send(JSON.stringify(response));
};

/**
 * podEventsHandler retrieves pod events and sends back as JSON format.
 * Also caches the events for later retrieval when K8s events expire.
 */
export const podEventsHandler: Handler = async (req, res) => {
  const { podname, podnamespace, runid, taskname } = req.query;
  if (!podname) {
    res.status(422).send('podname argument is required');
    return;
  }
  if (!podnamespace) {
    res.status(422).send('podnamespace argument is required');
    return;
  }
  const podName = decodeURIComponent(podname as string);
  const podNamespace = decodeURIComponent(podnamespace as string);
  const runId = runid ? decodeURIComponent(runid as string) : undefined;
  const taskName = taskname ? decodeURIComponent(taskname as string) : undefined;

  const [eventList, err] = await k8sHelper.listPodEvents(podName, podNamespace);

  // Extract events from K8s
  const events = err ? [] : extractEvents(eventList);

  // Get existing cached info
  const cached = await podEventsCache.getCachedPodInfo(podNamespace, podName);

  // If we have events from K8s, cache them (fire-and-forget)
  if (events.length > 0) {
    podEventsCache.savePodInfo(
      podName,
      podNamespace,
      cached?.status || null,
      events,
      runId,
      taskName,
    ).catch(() => {});
  }

  // Get merged cached events from all pods for this run/task
  const mergedCachedEvents = runId
    ? await podEventsCache.getMergedEvents(podNamespace, runId, taskName)
    : (cached?.events || []);

  // If K8s returned no events, fall back to merged cached events
  if (events.length === 0 && mergedCachedEvents.length > 0) {
    console.log('[podEventsHandler] Returning merged cached events for', podName, '(', mergedCachedEvents.length, 'events)');

    const cachedEventItems = mergedCachedEvents.map(e => ({
      type: e.type,
      reason: e.reason,
      message: e.message,
      firstTimestamp: e.firstTimestamp,
      lastTimestamp: e.lastTimestamp,
      count: e.count,
      source: { component: e.source },
      _cached: true,
    }));

    const stateHistory = runId
      ? await podEventsCache.getMergedStateHistory(podNamespace, runId, taskName)
      : (cached?.stateHistory || []);

    res.status(200).send(
      JSON.stringify({
        items: cachedEventItems,
        _cached: true,
        _cachedAt: cached?.lastUpdated || Date.now(),
        _stateHistory: stateHistory,
      }),
    );
    return;
  }

  if (err) {
    const { message, additionalInfo } = err;
    console.error(message, additionalInfo);
    res.status(500).send(message);
    return;
  }

  // Build response from fresh K8s events
  const eventResponse = JSON.parse(JSON.stringify(eventList));

  // Supplement with cached events from other pods
  if (runId && mergedCachedEvents.length > 0) {
    const freshKeys = new Set(
      events.map(e => `${e.type}:${e.reason}:${e.message.substring(0, 100)}`),
    );
    const supplementEvents = mergedCachedEvents
      .filter(e => !freshKeys.has(`${e.type}:${e.reason}:${e.message.substring(0, 100)}`))
      .map(e => ({
        type: e.type,
        reason: e.reason,
        message: e.message,
        firstTimestamp: e.firstTimestamp,
        lastTimestamp: e.lastTimestamp,
        count: e.count,
        source: { component: e.source },
        _cached: true,
      }));

    if (supplementEvents.length > 0) {
      eventResponse.items = [...(eventResponse.items || []), ...supplementEvents];
    }
  }

  // Attach merged state history
  if (runId) {
    const mergedHistory = await podEventsCache.getMergedStateHistory(podNamespace, runId, taskName);
    if (mergedHistory.length > 0) {
      eventResponse._stateHistory = mergedHistory;
    }
  } else {
    const updatedEventsCache = await podEventsCache.getCachedPodInfo(podNamespace, podName);
    if (updatedEventsCache?.stateHistory && updatedEventsCache.stateHistory.length > 0) {
      eventResponse._stateHistory = updatedEventsCache.stateHistory;
    }
  }

  res.status(200).send(JSON.stringify(eventResponse));
};

/**
 * podsByRunIdHandler lists pods by run ID label.
 * Also caches pod info AND events for later retrieval.
 * K8s API calls are parallelized to reduce total request time.
 */
export const podsByRunIdHandler: Handler = async (req, res) => {
  const { runid, podnamespace, taskname } = req.query;
  if (!runid) {
    res.status(422).send('runid argument is required');
    return;
  }
  if (!podnamespace) {
    res.status(422).send('podnamespace argument is required');
    return;
  }
  const runId = decodeURIComponent(runid as string);
  const podNamespace = decodeURIComponent(podnamespace as string);
  const taskName = taskname ? decodeURIComponent(taskname as string) : undefined;

  const [pods, err] = await k8sHelper.listPodsByRunId(runId, podNamespace, taskName);

  // Cache pod status AND fetch events for each pod found
  // Parallelized: all pods are processed concurrently instead of sequentially
  if (pods && pods.length > 0) {
    const cachePromises = pods.map(async (pod) => {
      const podName = (pod as any).metadata?.name;
      if (!podName) return;

      const status = extractPodStatus(pod);
      if (!status) return;

      const labels = (pod as any).metadata?.labels || {};
      const podTaskName =
        labels['pipelines.kubeflow.org/task_name'] ||
        labels['component'] ||
        taskName;

      // Fetch existing cache and events in parallel
      const [existing, eventResult] = await Promise.all([
        podEventsCache.getCachedPodInfo(podNamespace, podName),
        k8sHelper.listPodEvents(podName, podNamespace).catch(() => [undefined, { message: 'error' }] as const),
      ]);

      let events = existing?.events || [];
      const [eventList] = eventResult;
      if (eventList) {
        const freshEvents = extractEvents(eventList);
        events = freshEvents.length > 0 ? freshEvents : events;
      }

      await podEventsCache.savePodInfo(
        podName,
        podNamespace,
        status,
        events,
        runId,
        podTaskName,
      );
    });

    // Fire-and-forget caching: don't block the response
    Promise.all(cachePromises).catch(() => {});
  }

  // If no pods found in K8s, try to return cached pod info
  if ((!pods || pods.length === 0) && !err) {
    const cachedPod = await podEventsCache.getCachedPodInfoByRunId(podNamespace, runId, taskName);
    if (cachedPod) {
      console.log('[podsByRunIdHandler] Returning cached pod info for run', runId);
      res.status(200).send(
        JSON.stringify([
          {
            metadata: { name: cachedPod.podName, namespace: cachedPod.namespace },
            status: cachedPod.status
              ? {
                  phase: cachedPod.status.phase,
                  message: cachedPod.status.message,
                  reason: cachedPod.status.reason,
                  podIP: cachedPod.status.podIP,
                  containerStatuses: cachedPod.status.containerStatuses,
                }
              : undefined,
            spec: cachedPod.status ? { nodeName: cachedPod.status.nodeName } : undefined,
            _cached: true,
            _cachedAt: cachedPod.lastUpdated,
          },
        ]),
      );
      return;
    }
  }

  if (err) {
    const { message, additionalInfo } = err;
    console.error('[podsByRunIdHandler] Error:', message, additionalInfo);
    res.status(500).send(message);
    return;
  }

  res.status(200).send(JSON.stringify(pods));
};

/**
 * cachedPodInfoHandler retrieves cached pod info by run ID.
 */
export const cachedPodInfoHandler: Handler = async (req, res) => {
  const { podname, podnamespace, taskname } = req.query;

  const podNamespace = podnamespace ? decodeURIComponent(podnamespace as string) : undefined;

  if (!podNamespace) {
    res.status(422).send('podnamespace argument is required');
    return;
  }

  // If pod name provided, get by pod name
  if (podname) {
    const podName = decodeURIComponent(podname as string);
    const cached = await podEventsCache.getCachedPodInfo(podNamespace, podName);

    if (!cached) {
      res.status(404).send('No cached data found');
      return;
    }

    res.status(200).send(JSON.stringify(cached));
    return;
  }

  // If run ID provided, search by run ID
  if (req.query.runid) {
    const runId = decodeURIComponent(req.query.runid as string);
    const taskName = taskname ? decodeURIComponent(taskname as string) : undefined;

    const cached = await podEventsCache.getCachedPodInfoByRunId(podNamespace, runId, taskName);

    if (!cached) {
      res.status(404).send('No cached data found');
      return;
    }

    res.status(200).send(JSON.stringify(cached));
    return;
  }

  res.status(422).send('Either podname or runid argument is required');
};
