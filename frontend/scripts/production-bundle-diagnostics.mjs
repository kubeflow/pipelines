/*
 * Copyright 2026 The Kubeflow Authors
 * Licensed under the Apache License, Version 2.0 (the "License");
 * You may obtain a copy of the License at https://www.apache.org/licenses/LICENSE-2.0
 */
import { mkdirSync, writeFileSync } from 'node:fs';
import { resolve } from 'node:path';

// Synchronous checkpoints survive node:test cancellation of an awaited browser command.
export function startupDiagnostics(directory, identity) {
  mkdirSync(directory, { recursive: true });
  const report = {
    ...identity,
    status: 'running',
    stage: 'initializing',
    events: [],
    pageErrors: [],
  };
  const save = () =>
    writeFileSync(
      resolve(directory, 'production-startup.json'),
      `${JSON.stringify(report, null, 2)}\n`,
    );
  return {
    report,
    stage(name) {
      report.stage = name;
      report.events.push({ stage: name, at: new Date().toISOString() });
      save();
    },
    pageError(error) {
      report.pageErrors.push(String(error));
      save();
    },
    fail(error) {
      if (!report.failure)
        report.failure = {
          stage: report.stage,
          error: String(error),
          at: new Date().toISOString(),
        };
      report.status = 'failed';
      save();
    },
    pass() {
      if (!report.failure) report.status = 'passed';
      save();
    },
  };
}
