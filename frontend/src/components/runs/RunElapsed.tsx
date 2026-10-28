// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

import { useEffect, useState } from 'react';
import { V2Run, V2RuntimeState } from 'src/apisv2/run';
import { hasFinishedV2 } from 'src/lib/StatusUtils';
import { getRunDurationFromRunV2 } from 'src/lib/Utils';

function timestamp(value?: Date): number | undefined {
  const time = value && new Date(value).getTime();
  return time !== undefined && Number.isFinite(time) && time > 0 ? time : undefined;
}

/** Keep the clock local to the header so ticking does not rerender the graph. */
export function RunElapsed({ run }: { run: V2Run }) {
  const [now, setNow] = useState(Date.now);
  const active = run.state === V2RuntimeState.RUNNING || run.state === V2RuntimeState.CANCELING;
  // External sync: the timer exists only while the run is active.
  useEffect(() => {
    if (!active) return;
    const timer = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(timer);
  }, [active]);

  const starts = (run.state_history || [])
    .filter((status) => status.state === V2RuntimeState.RUNNING)
    .map((status) => timestamp(status.update_time))
    .filter((time): time is number => time !== undefined);
  const started = starts.length ? Math.min(...starts) : undefined;
  const start = started ?? timestamp(run.created_at);
  const end = hasFinishedV2(run.state) ? timestamp(run.finished_at) : active ? now : undefined;
  let text = 'Elapsed unavailable';
  if (run.state === V2RuntimeState.PENDING) {
    text = 'Not started';
  } else if (start !== undefined && end !== undefined && end >= start) {
    const duration = getRunDurationFromRunV2({
      created_at: new Date(start),
      finished_at: new Date(end),
    });
    text = `${started === undefined ? 'Since created' : 'Elapsed'} ${duration}`;
  }
  return <span className='kfp-inspection-run-elapsed'>{text}</span>;
}
