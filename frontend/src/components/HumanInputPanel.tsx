/*
 * Copyright 2024 The Kubeflow Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import {
  Alert,
  Box,
  Button,
  CircularProgress,
  FormControl,
  FormHelperText,
  InputLabel,
  MenuItem,
  Select,
  TextField,
  Typography,
} from '@mui/material';
import * as React from 'react';
import { useCallback, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { queryKeys } from 'src/hooks/queryKeys';

/**
 * HUMAN_INPUT_SENTINEL_IMAGE is the container image string that KFP uses to
 * mark a task as a human-input (suspend) node in the pipeline spec.
 * Must match the constant in sdk/python/kfp/dsl/human_input.py and
 * backend/src/v2/compiler/argocompiler/argo.go.
 */
export const HUMAN_INPUT_SENTINEL_IMAGE = 'kfp://human-input';

/** Per-parameter metadata returned by the GET intermediateInputs endpoint. */
export interface IntermediateParamStatus {
  description?: string;
  default?: string;
  /** Static or pre-resolved enum choices.  Empty → free-text input. */
  enum?: string[];
  /** Value already supplied (if the node was resumed previously). */
  current_value?: string | null;
}

/** Response body from GET /apis/v2beta1/runs/{run_id}/nodes/{node_id}/intermediateInputs */
export interface IntermediateInputsStatus {
  suspended: boolean;
  parameters?: Record<string, IntermediateParamStatus>;
}

// ---------------------------------------------------------------------------
// API helpers
// ---------------------------------------------------------------------------

const V2BETA1_PREFIX = 'apis/v2beta1';

/**
 * Fetches the current intermediate-input status for a suspend node.
 */
async function fetchIntermediateInputsStatus(
  runId: string,
  nodeDisplayName: string,
): Promise<IntermediateInputsStatus> {
  const url = `${V2BETA1_PREFIX}/runs/${encodeURIComponent(runId)}/nodes/${encodeURIComponent(nodeDisplayName)}/intermediateInputs`;
  const resp = await fetch(url, { credentials: 'same-origin' });
  const text = await resp.text();
  if (!resp.ok) {
    throw new Error(`Failed to fetch intermediate inputs status: ${text}`);
  }
  return JSON.parse(text) as IntermediateInputsStatus;
}

/**
 * Submits human-supplied parameter values to a paused suspend node and
 * resumes the workflow.
 */
async function submitIntermediateInputs(
  runId: string,
  nodeDisplayName: string,
  parameters: Record<string, string>,
): Promise<void> {
  const url = `${V2BETA1_PREFIX}/runs/${encodeURIComponent(runId)}/nodes/${encodeURIComponent(nodeDisplayName)}:setIntermediateInputs`;
  const resp = await fetch(url, {
    credentials: 'same-origin',
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ parameters }),
  });
  const text = await resp.text();
  if (!resp.ok) {
    // Try to surface a clean error message from the backend JSON response.
    let message = text;
    try {
      const body = JSON.parse(text) as { error_message?: string };
      if (body.error_message) {
        message = body.error_message;
      }
    } catch {
      // Keep the raw text if parsing fails.
    }
    throw new Error(message);
  }
}

// ---------------------------------------------------------------------------
// Sub-components
// ---------------------------------------------------------------------------

interface ParameterFieldProps {
  name: string;
  meta: IntermediateParamStatus;
  value: string;
  onChange: (name: string, value: string) => void;
  disabled: boolean;
}

/**
 * Renders a single form field for one human-input parameter.
 *
 * - When the template declares static enum choices those are shown as a
 *   dropdown `<Select>`.
 * - Otherwise a free-text `<TextField>` is rendered.
 */
function ParameterField({ name, meta, value, onChange, disabled }: ParameterFieldProps) {
  const enumChoices = meta.enum && meta.enum.length > 0 ? meta.enum : undefined;

  const helperText = [
    meta.description,
    meta.default !== undefined && meta.default !== null ? `Default: ${meta.default}` : undefined,
  ]
    .filter(Boolean)
    .join(' · ');

  if (enumChoices && enumChoices.length > 0) {
    const selectId = `human-input-select-${name}`;
    return (
      <FormControl variant='outlined' size='small' fullWidth sx={{ mb: 2 }} disabled={disabled}>
        <InputLabel id={`${selectId}-label`}>{name}</InputLabel>
        <Select
          labelId={`${selectId}-label`}
          id={selectId}
          value={value}
          label={name}
          onChange={(e) => onChange(name, e.target.value as string)}
          inputProps={{ 'aria-label': name }}
        >
          {enumChoices.map((choice) => (
            <MenuItem key={choice} value={choice}>
              {choice}
            </MenuItem>
          ))}
        </Select>
        {helperText && <FormHelperText>{helperText}</FormHelperText>}
      </FormControl>
    );
  }

  return (
    <TextField
      label={name}
      value={value}
      onChange={(e) => onChange(name, e.target.value)}
      variant='outlined'
      size='small'
      fullWidth
      disabled={disabled}
      helperText={helperText || undefined}
      sx={{ mb: 2 }}
      inputProps={{ 'aria-label': name }}
    />
  );
}

// ---------------------------------------------------------------------------
// Main component
// ---------------------------------------------------------------------------

export interface HumanInputPanelProps {
  /** KFP run identifier. */
  runId: string;
  /**
   * The display name of the suspend node in the Argo workflow.
   * Corresponds to the task name used in the pipeline DSL.
   */
  nodeDisplayName: string;
  /**
   * Parameter metadata parsed from the container args in the pipeline spec.
   * Keys are parameter names; values hold description/default/enum extracted
   * from the JSON encoded in args[0] of the sentinel container.
   */
  paramMetaFromSpec?: Record<string, IntermediateParamStatus>;
}

/**
 * HumanInputPanel allows a user to supply parameter values to a suspended
 * (human-input) task in a running KFP pipeline.
 *
 * It polls the backend to detect when the node enters the suspended state,
 * renders one form control per declared parameter, and submits the values
 * so the workflow can continue.
 */
export function HumanInputPanel({
  runId,
  nodeDisplayName,
  paramMetaFromSpec,
}: HumanInputPanelProps) {
  const queryClient = useQueryClient();

  // Poll for suspend-node status so the UI stays current as the workflow
  // advances.  The interval is stopped once the node is no longer suspended.
  const {
    data: statusData,
    isError: statusError,
    error: statusFetchError,
  } = useQuery<IntermediateInputsStatus, Error>({
    queryKey: queryKeys.intermediateInputsStatus(runId, nodeDisplayName),
    queryFn: () => fetchIntermediateInputsStatus(runId, nodeDisplayName),
    // Refresh every 5 s while the panel is visible.
    refetchInterval: 5000,
  });

  // Merge spec-level metadata with the live status from the backend.  The
  // backend is authoritative for the set of declared parameters; the spec
  // metadata enriches it with description/default/enum when available.
  const mergedParams: Record<string, IntermediateParamStatus> = {};
  const liveParams = statusData?.parameters ?? {};
  const specParams = paramMetaFromSpec ?? {};
  const allParamNames = new Set([...Object.keys(liveParams), ...Object.keys(specParams)]);
  for (const name of allParamNames) {
    mergedParams[name] = { ...specParams[name], ...liveParams[name] };
  }

  const [editedValues, setEditedValues] = useState<Record<string, string>>({});
  const formValues = Object.fromEntries(
    Object.entries(mergedParams).map(([name, meta]) => [
      name,
      editedValues[name] ?? meta.current_value ?? meta.default ?? '',
    ]),
  );

  const handleFieldChange = useCallback((name: string, value: string) => {
    setEditedValues((prev) => ({ ...prev, [name]: value }));
  }, []);

  const submitMutation = useMutation<void, Error, Record<string, string>>({
    mutationFn: (values) => submitIntermediateInputs(runId, nodeDisplayName, values),
    onSuccess: () => {
      // Invalidate the status query so the panel reflects the new state
      // (Succeeded) without waiting for the next poll interval.
      void queryClient.invalidateQueries({
        queryKey: queryKeys.intermediateInputsStatus(runId, nodeDisplayName),
      });
    },
  });

  const isSuspended = statusData?.suspended === true;
  const isSubmitting = submitMutation.isPending;
  const isSubmitted = submitMutation.isSuccess;

  const handleSubmit = () => {
    if (!isSuspended || isSubmitting || isSubmitted) {
      return;
    }
    submitMutation.mutate(formValues);
  };

  const paramNames = Object.keys(mergedParams);

  return (
    <Box sx={{ p: 2 }}>
      <Typography variant='h6' sx={{ mb: 1 }}>
        Human Input Required
      </Typography>

      {/* Node status indicator */}
      {!statusData && !statusError && (
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 2 }}>
          <CircularProgress size={16} />
          <Typography variant='body2'>Loading node status…</Typography>
        </Box>
      )}
      {statusError && (
        <Alert severity='warning' sx={{ mb: 2 }}>
          Could not load node status: {statusFetchError?.message ?? 'unknown error'}
        </Alert>
      )}
      {statusData && !isSuspended && !isSubmitted && (
        <Alert severity='info' sx={{ mb: 2 }}>
          This task has not yet reached the suspended state. The form will become active once the
          workflow pauses here.
        </Alert>
      )}
      {isSuspended && !isSubmitted && (
        <Alert severity='warning' sx={{ mb: 2 }}>
          The pipeline is paused at this step. Please provide the required inputs below and click{' '}
          <strong>Submit &amp; Resume</strong>.
        </Alert>
      )}
      {isSubmitted && (
        <Alert severity='success' sx={{ mb: 2 }}>
          Inputs submitted successfully. The pipeline will continue.
        </Alert>
      )}

      {/* Submission error */}
      {submitMutation.isError && (
        <Alert severity='error' sx={{ mb: 2 }} role='alert'>
          Failed to submit: {submitMutation.error?.message ?? 'unknown error'}
        </Alert>
      )}

      {/* Parameter form */}
      {paramNames.length > 0 && (
        <Box component='form' noValidate sx={{ mt: 1 }} aria-label='Human input parameters'>
          {paramNames.map((name) => {
            const meta = mergedParams[name];
            return (
              <ParameterField
                key={name}
                name={name}
                meta={meta}
                value={formValues[name] ?? ''}
                onChange={handleFieldChange}
                disabled={!isSuspended || isSubmitting || isSubmitted}
              />
            );
          })}

          <Button
            variant='contained'
            color='primary'
            onClick={handleSubmit}
            disabled={!isSuspended || isSubmitting || isSubmitted}
            startIcon={isSubmitting ? <CircularProgress size={16} color='inherit' /> : undefined}
            aria-label='Submit and resume the pipeline'
          >
            {isSubmitting ? 'Submitting…' : isSubmitted ? 'Submitted' : 'Submit & Resume'}
          </Button>
        </Box>
      )}

      {paramNames.length === 0 && statusData && (
        <Typography variant='body2' color='text.secondary'>
          No parameters declared for this human-input step.
        </Typography>
      )}
    </Box>
  );
}
