/*
 * Copyright 2021 The Kubeflow Authors
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

import * as React from 'react';
import { useState } from 'react';
import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControl,
  FormControlLabel,
  Radio,
  RadioGroup,
  TextField,
  Checkbox,
} from '@mui/material';
import { V2beta1TaskScope } from 'src/apisv2beta1/run';

export type TaskActionKind = 'clear' | 'markSuccess';

interface TaskActionDialogProps {
  open: boolean;
  kind: TaskActionKind;
  taskName: string;
  onClose: () => void;
  onConfirm: (
    scope: V2beta1TaskScope,
    extra: { invalidateCache?: boolean; comment?: string },
  ) => void;
}

const SCOPE_OPTIONS: { value: V2beta1TaskScope; label: string; help: string }[] = [
  { value: V2beta1TaskScope.TASK_ONLY, label: 'Task only', help: 'Only this task is affected.' },
  {
    value: V2beta1TaskScope.DOWNSTREAM,
    label: 'Downstream',
    help: 'This task and everything that depends on it.',
  },
  {
    value: V2beta1TaskScope.UPSTREAM,
    label: 'Upstream',
    help: 'This task and everything it depends on.',
  },
  {
    value: V2beta1TaskScope.UPSTREAM_DOWNSTREAM,
    label: 'Upstream + Downstream',
    help: 'The whole connected subgraph.',
  },
];

export default function TaskActionDialog({
  open,
  kind,
  taskName,
  onClose,
  onConfirm,
}: TaskActionDialogProps) {
  const [scope, setScope] = useState<V2beta1TaskScope>(V2beta1TaskScope.TASK_ONLY);
  const [invalidateCache, setInvalidateCache] = useState(false);
  const [comment, setComment] = useState('');

  const isMarkSuccess = kind === 'markSuccess';
  const commentInvalid = isMarkSuccess && comment.trim().length === 0;

  return (
    <Dialog open={open} onClose={onClose} maxWidth='xs' fullWidth>
      <DialogTitle>
        {isMarkSuccess ? `Mark "${taskName}" as succeeded?` : `Clear and re-run "${taskName}"?`}
      </DialogTitle>
      <DialogContent>
        {isMarkSuccess && (
          <p>
            This does not re-run the task — it tells KFP to treat it as done, without checking that
            any expected outputs actually exist. Downstream tasks may fail if they need those
            outputs.
          </p>
        )}
        <FormControl component='fieldset'>
          <RadioGroup value={scope} onChange={(e) => setScope(e.target.value as V2beta1TaskScope)}>
            {SCOPE_OPTIONS.map((opt) => (
              <FormControlLabel
                key={opt.value}
                value={opt.value}
                control={<Radio color='primary' />}
                label={`${opt.label} — ${opt.help}`}
              />
            ))}
          </RadioGroup>
        </FormControl>

        {!isMarkSuccess && (
          <FormControlLabel
            control={
              <Checkbox
                checked={invalidateCache}
                onChange={(e) => setInvalidateCache(e.target.checked)}
              />
            }
            label='Ignore cache (force a real re-run instead of reusing a cached result)'
          />
        )}

        {isMarkSuccess && (
          <TextField
            label='Reason for override (required)'
            required
            fullWidth
            multiline
            minRows={2}
            value={comment}
            onChange={(e) => setComment(e.target.value)}
            error={commentInvalid}
            helperText={commentInvalid ? 'A reason is required for audit purposes.' : ' '}
          />
        )}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Cancel</Button>
        <Button
          color='primary'
          variant='contained'
          disabled={commentInvalid}
          onClick={() => onConfirm(scope, { invalidateCache, comment })}
        >
          Confirm
        </Button>
      </DialogActions>
    </Dialog>
  );
}
