/*
 * Copyright 2023 The Kubeflow Authors
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

import RunUtils from './RunUtils';
import {
    ApiResourceReference,
    ApiResourceType,
    ApiRun,
    ApiRunDetail,
    ApiRelationship,
} from '../apis/run';
import { ApiJob } from '../apis/job';
import { ApiPipelineVersion } from '../apis/pipeline';
import { ApiExperiment } from '../apis/experiment';
import WorkflowParser from './WorkflowParser';
import { logger } from './Utils';

describe('RunUtils', () => {
    beforeEach(() => {
        jest.restoreAllMocks();
    });

    describe('getParametersFromRuntime', () => {
        it('returns empty array for undefined runtime', () => {
            expect(RunUtils.getParametersFromRuntime(undefined)).toEqual([]);
        });

        it('returns empty array for runtime without workflow_manifest', () => {
            expect(RunUtils.getParametersFromRuntime({})).toEqual([]);
        });

        it('returns parameters from valid workflow manifest', () => {
            const mockParams = [{ name: 'param1', value: 'value1' }];
            const getParametersSpy = jest.spyOn(WorkflowParser, 'getParameters').mockReturnValue(mockParams);

            const runtime = {
                workflow_manifest: JSON.stringify({ spec: { arguments: { parameters: mockParams } } }),
            };
            expect(RunUtils.getParametersFromRuntime(runtime)).toEqual(mockParams);
            expect(getParametersSpy).toHaveBeenCalled();
        });

        it('returns empty array and logs error for invalid JSON', () => {
            const loggerSpy = jest.spyOn(logger, 'error').mockImplementation(() => { });
            const runtime = { workflow_manifest: 'not valid json' };
            expect(RunUtils.getParametersFromRuntime(runtime)).toEqual([]);
            expect(loggerSpy).toHaveBeenCalled();
        });
    });

    describe('getParametersFromRun', () => {
        it('returns parameters from run pipeline_runtime', () => {
            const mockParams = [{ name: 'param1', value: 'value1' }];
            const getParametersSpy = jest.spyOn(WorkflowParser, 'getParameters').mockReturnValue(mockParams);

            const run: ApiRunDetail = {
                pipeline_runtime: {
                    workflow_manifest: JSON.stringify({ spec: {} }),
                },
            };
            expect(RunUtils.getParametersFromRun(run)).toEqual(mockParams);
            expect(getParametersSpy).toHaveBeenCalled();
        });
    });

    describe('getPipelineId', () => {
        it('returns null for undefined run', () => {
            expect(RunUtils.getPipelineId(undefined)).toBeNull();
        });

        it('returns null for run without pipeline_spec', () => {
            const run: ApiRun = {};
            expect(RunUtils.getPipelineId(run)).toBeNull();
        });

        it('returns null for run with pipeline_spec but no pipeline_id', () => {
            const run: ApiRun = { pipeline_spec: {} };
            expect(RunUtils.getPipelineId(run)).toBeNull();
        });

        it('returns pipeline_id from run', () => {
            const run: ApiRun = { pipeline_spec: { pipeline_id: 'test-id' } };
            expect(RunUtils.getPipelineId(run)).toBe('test-id');
        });

        it('returns pipeline_id from job', () => {
            const job: ApiJob = { pipeline_spec: { pipeline_id: 'job-pipeline-id' } };
            expect(RunUtils.getPipelineId(job)).toBe('job-pipeline-id');
        });
    });

    describe('getPipelineName', () => {
        it('returns null for undefined run', () => {
            expect(RunUtils.getPipelineName(undefined)).toBeNull();
        });

        it('returns null for run without pipeline_spec', () => {
            const run: ApiRun = {};
            expect(RunUtils.getPipelineName(run)).toBeNull();
        });

        it('returns null for run with pipeline_spec but no pipeline_name', () => {
            const run: ApiRun = { pipeline_spec: {} };
            expect(RunUtils.getPipelineName(run)).toBeNull();
        });

        it('returns pipeline_name from run', () => {
            const run: ApiRun = { pipeline_spec: { pipeline_name: 'test-name' } };
            expect(RunUtils.getPipelineName(run)).toBe('test-name');
        });
    });

    describe('getPipelineVersionId', () => {
        it('returns null for undefined run', () => {
            expect(RunUtils.getPipelineVersionId(undefined)).toBeNull();
        });

        it('returns null for run without resource_references', () => {
            const run: ApiRun = {};
            expect(RunUtils.getPipelineVersionId(run)).toBeNull();
        });

        it('returns null for run with empty resource_references', () => {
            const run: ApiRun = { resource_references: [] };
            expect(RunUtils.getPipelineVersionId(run)).toBeNull();
        });

        it('returns null for run with non-pipeline-version references', () => {
            const run: ApiRun = {
                resource_references: [{ key: { type: ApiResourceType.EXPERIMENT, id: 'exp-id' } }],
            };
            expect(RunUtils.getPipelineVersionId(run)).toBeNull();
        });

        it('returns pipeline version id from run', () => {
            const run: ApiRun = {
                resource_references: [
                    { key: { type: ApiResourceType.PIPELINEVERSION, id: 'version-id' } },
                ],
            };
            expect(RunUtils.getPipelineVersionId(run)).toBe('version-id');
        });
    });

    describe('getPipelineIdFromApiPipelineVersion', () => {
        it('returns undefined for undefined pipelineVersion', () => {
            expect(RunUtils.getPipelineIdFromApiPipelineVersion(undefined)).toBeUndefined();
        });

        it('returns undefined for pipelineVersion without resource_references', () => {
            const pipelineVersion: ApiPipelineVersion = {};
            expect(RunUtils.getPipelineIdFromApiPipelineVersion(pipelineVersion)).toBeUndefined();
        });

        it('returns undefined for pipelineVersion with empty resource_references', () => {
            const pipelineVersion: ApiPipelineVersion = { resource_references: [] };
            expect(RunUtils.getPipelineIdFromApiPipelineVersion(pipelineVersion)).toBeUndefined();
        });

        it('returns undefined for pipelineVersion with non-pipeline references', () => {
            const pipelineVersion: ApiPipelineVersion = {
                resource_references: [{ key: { type: ApiResourceType.EXPERIMENT, id: 'exp-id' } }],
            };
            expect(RunUtils.getPipelineIdFromApiPipelineVersion(pipelineVersion)).toBeUndefined();
        });

        it('returns pipeline id from pipelineVersion', () => {
            const pipelineVersion: ApiPipelineVersion = {
                resource_references: [{ key: { type: ApiResourceType.PIPELINE, id: 'pipeline-id' } }],
            };
            expect(RunUtils.getPipelineIdFromApiPipelineVersion(pipelineVersion)).toBe('pipeline-id');
        });
    });

    describe('getWorkflowManifest', () => {
        it('returns null for undefined run', () => {
            expect(RunUtils.getWorkflowManifest(undefined)).toBeNull();
        });

        it('returns null for run without pipeline_spec', () => {
            const run: ApiRun = {};
            expect(RunUtils.getWorkflowManifest(run)).toBeNull();
        });

        it('returns null for run with pipeline_spec but no workflow_manifest', () => {
            const run: ApiRun = { pipeline_spec: {} };
            expect(RunUtils.getWorkflowManifest(run)).toBeNull();
        });

        it('returns workflow_manifest from run', () => {
            const manifest = '{"some": "manifest"}';
            const run: ApiRun = { pipeline_spec: { workflow_manifest: manifest } };
            expect(RunUtils.getWorkflowManifest(run)).toBe(manifest);
        });
    });

    describe('getAllExperimentReferences', () => {
        it('returns empty array for undefined run', () => {
            expect(RunUtils.getAllExperimentReferences(undefined)).toEqual([]);
        });

        it('returns empty array for run without resource_references', () => {
            const run: ApiRun = {};
            expect(RunUtils.getAllExperimentReferences(run)).toEqual([]);
        });

        it('returns empty array for run with no experiment references', () => {
            const run: ApiRun = {
                resource_references: [{ key: { type: ApiResourceType.PIPELINE, id: 'pipeline-id' } }],
            };
            expect(RunUtils.getAllExperimentReferences(run)).toEqual([]);
        });

        it('returns only experiment references', () => {
            const expRef: ApiResourceReference = {
                key: { type: ApiResourceType.EXPERIMENT, id: 'exp-id' },
                name: 'exp-name',
            };
            const run: ApiRun = {
                resource_references: [
                    { key: { type: ApiResourceType.PIPELINE, id: 'pipeline-id' } },
                    expRef,
                ],
            };
            expect(RunUtils.getAllExperimentReferences(run)).toEqual([expRef]);
        });

        it('returns multiple experiment references', () => {
            const expRef1: ApiResourceReference = {
                key: { type: ApiResourceType.EXPERIMENT, id: 'exp-id-1' },
            };
            const expRef2: ApiResourceReference = {
                key: { type: ApiResourceType.EXPERIMENT, id: 'exp-id-2' },
            };
            const run: ApiRun = {
                resource_references: [expRef1, expRef2],
            };
            expect(RunUtils.getAllExperimentReferences(run)).toEqual([expRef1, expRef2]);
        });
    });

    describe('getFirstExperimentReference', () => {
        it('returns null for undefined run', () => {
            expect(RunUtils.getFirstExperimentReference(undefined)).toBeNull();
        });

        it('returns null for run with no experiment references', () => {
            const run: ApiRun = { resource_references: [] };
            expect(RunUtils.getFirstExperimentReference(run)).toBeNull();
        });

        it('returns first experiment reference', () => {
            const expRef: ApiResourceReference = {
                key: { type: ApiResourceType.EXPERIMENT, id: 'exp-id' },
                name: 'exp-name',
            };
            const run: ApiRun = { resource_references: [expRef] };
            expect(RunUtils.getFirstExperimentReference(run)).toEqual(expRef);
        });
    });

    describe('getFirstExperimentReferenceId', () => {
        it('returns null for undefined run', () => {
            expect(RunUtils.getFirstExperimentReferenceId(undefined)).toBeNull();
        });

        it('returns null for run with no experiment references', () => {
            const run: ApiRun = { resource_references: [] };
            expect(RunUtils.getFirstExperimentReferenceId(run)).toBeNull();
        });

        it('returns null for experiment reference without key.id', () => {
            const run: ApiRun = {
                resource_references: [{ key: { type: ApiResourceType.EXPERIMENT } }],
            };
            expect(RunUtils.getFirstExperimentReferenceId(run)).toBeNull();
        });

        it('returns experiment reference id', () => {
            const run: ApiRun = {
                resource_references: [{ key: { type: ApiResourceType.EXPERIMENT, id: 'exp-id' } }],
            };
            expect(RunUtils.getFirstExperimentReferenceId(run)).toBe('exp-id');
        });
    });

    describe('getFirstExperimentReferenceName', () => {
        it('returns null for undefined run', () => {
            expect(RunUtils.getFirstExperimentReferenceName(undefined)).toBeNull();
        });

        it('returns null for run with no experiment references', () => {
            const run: ApiRun = { resource_references: [] };
            expect(RunUtils.getFirstExperimentReferenceName(run)).toBeNull();
        });

        it('returns null for experiment reference without name', () => {
            const run: ApiRun = {
                resource_references: [{ key: { type: ApiResourceType.EXPERIMENT, id: 'exp-id' } }],
            };
            expect(RunUtils.getFirstExperimentReferenceName(run)).toBeNull();
        });

        it('returns experiment reference name', () => {
            const run: ApiRun = {
                resource_references: [
                    { key: { type: ApiResourceType.EXPERIMENT, id: 'exp-id' }, name: 'exp-name' },
                ],
            };
            expect(RunUtils.getFirstExperimentReferenceName(run)).toBe('exp-name');
        });
    });

    describe('getNamespaceReferenceName', () => {
        it('returns undefined for undefined experiment', () => {
            expect(RunUtils.getNamespaceReferenceName(undefined)).toBeUndefined();
        });

        it('returns undefined for experiment without resource_references', () => {
            const experiment: ApiExperiment = {};
            expect(RunUtils.getNamespaceReferenceName(experiment)).toBeUndefined();
        });

        it('returns undefined for experiment with no namespace reference', () => {
            const experiment: ApiExperiment = {
                resource_references: [{ key: { type: ApiResourceType.EXPERIMENT, id: 'exp-id' } }],
            };
            expect(RunUtils.getNamespaceReferenceName(experiment)).toBeUndefined();
        });

        it('returns undefined for namespace reference without OWNER relationship', () => {
            const experiment: ApiExperiment = {
                resource_references: [
                    {
                        key: { type: ApiResourceType.NAMESPACE, id: 'ns-id' },
                        relationship: ApiRelationship.CREATOR,
                    },
                ],
            };
            expect(RunUtils.getNamespaceReferenceName(experiment)).toBeUndefined();
        });

        it('returns namespace id for valid namespace reference', () => {
            const experiment: ApiExperiment = {
                resource_references: [
                    {
                        key: { type: ApiResourceType.NAMESPACE, id: 'my-namespace' },
                        relationship: ApiRelationship.OWNER,
                    },
                ],
            };
            expect(RunUtils.getNamespaceReferenceName(experiment)).toBe('my-namespace');
        });
    });

    describe('runsToMetricMetadataMap', () => {
        it('returns empty map for empty runs array', () => {
            expect(RunUtils.runsToMetricMetadataMap([])).toEqual(new Map());
        });

        it('returns empty map for runs without metrics', () => {
            const runs: ApiRun[] = [{ id: 'run-1' }, { id: 'run-2' }];
            expect(RunUtils.runsToMetricMetadataMap(runs)).toEqual(new Map());
        });

        it('returns empty map for runs with empty metrics array', () => {
            const runs: ApiRun[] = [{ id: 'run-1', metrics: [] }];
            expect(RunUtils.runsToMetricMetadataMap(runs)).toEqual(new Map());
        });

        it('ignores metrics without name', () => {
            const runs: ApiRun[] = [{ id: 'run-1', metrics: [{ number_value: 0.5 }] }];
            expect(RunUtils.runsToMetricMetadataMap(runs)).toEqual(new Map());
        });

        it('ignores metrics with undefined number_value', () => {
            const runs: ApiRun[] = [{ id: 'run-1', metrics: [{ name: 'accuracy' }] }];
            expect(RunUtils.runsToMetricMetadataMap(runs)).toEqual(new Map());
        });

        it('ignores metrics with NaN number_value', () => {
            const runs: ApiRun[] = [{ id: 'run-1', metrics: [{ name: 'accuracy', number_value: NaN }] }];
            expect(RunUtils.runsToMetricMetadataMap(runs)).toEqual(new Map());
        });

        it('aggregates single metric correctly', () => {
            const runs: ApiRun[] = [
                { id: 'run-1', metrics: [{ name: 'accuracy', number_value: 0.85 }] },
            ];
            const result = RunUtils.runsToMetricMetadataMap(runs);
            expect(result.get('accuracy')).toEqual({
                count: 1,
                maxValue: 0.85,
                minValue: 0.85,
                name: 'accuracy',
            });
        });

        it('aggregates same metric across multiple runs', () => {
            const runs: ApiRun[] = [
                { id: 'run-1', metrics: [{ name: 'accuracy', number_value: 0.8 }] },
                { id: 'run-2', metrics: [{ name: 'accuracy', number_value: 0.9 }] },
                { id: 'run-3', metrics: [{ name: 'accuracy', number_value: 0.7 }] },
            ];
            const result = RunUtils.runsToMetricMetadataMap(runs);
            expect(result.get('accuracy')).toEqual({
                count: 3,
                maxValue: 0.9,
                minValue: 0.7,
                name: 'accuracy',
            });
        });

        it('handles multiple different metrics', () => {
            const runs: ApiRun[] = [
                {
                    id: 'run-1',
                    metrics: [
                        { name: 'accuracy', number_value: 0.85 },
                        { name: 'loss', number_value: 0.15 },
                    ],
                },
            ];
            const result = RunUtils.runsToMetricMetadataMap(runs);
            expect(result.size).toBe(2);
            expect(result.get('accuracy')).toEqual({
                count: 1,
                maxValue: 0.85,
                minValue: 0.85,
                name: 'accuracy',
            });
            expect(result.get('loss')).toEqual({
                count: 1,
                maxValue: 0.15,
                minValue: 0.15,
                name: 'loss',
            });
        });

        it('handles null run in array', () => {
            const runs = [null, { id: 'run-1', metrics: [{ name: 'accuracy', number_value: 0.9 }] }];
            const result = RunUtils.runsToMetricMetadataMap(runs as ApiRun[]);
            expect(result.get('accuracy')).toEqual({
                count: 1,
                maxValue: 0.9,
                minValue: 0.9,
                name: 'accuracy',
            });
        });
    });

    describe('extractMetricMetadata', () => {
        it('returns empty array for empty runs', () => {
            expect(RunUtils.extractMetricMetadata([])).toEqual([]);
        });

        it('returns array of metric metadata from runs', () => {
            const runs: ApiRun[] = [
                { id: 'run-1', metrics: [{ name: 'accuracy', number_value: 0.85 }] },
            ];
            const result = RunUtils.extractMetricMetadata(runs);
            expect(result).toEqual([
                {
                    count: 1,
                    maxValue: 0.85,
                    minValue: 0.85,
                    name: 'accuracy',
                },
            ]);
        });
    });

    describe('getRecurringRunId', () => {
        it('returns empty string for undefined run', () => {
            expect(RunUtils.getRecurringRunId(undefined)).toBe('');
        });

        it('returns empty string for run without resource_references', () => {
            const run: ApiRun = {};
            expect(RunUtils.getRecurringRunId(run)).toBe('');
        });

        it('returns empty string for run with no JOB reference', () => {
            const run: ApiRun = {
                resource_references: [{ key: { type: ApiResourceType.EXPERIMENT, id: 'exp-id' } }],
            };
            expect(RunUtils.getRecurringRunId(run)).toBe('');
        });

        it('returns recurring run id from JOB reference', () => {
            const run: ApiRun = {
                resource_references: [{ key: { type: ApiResourceType.JOB, id: 'job-id' } }],
            };
            expect(RunUtils.getRecurringRunId(run)).toBe('job-id');
        });

        it('returns empty string for JOB reference without id', () => {
            const run: ApiRun = {
                resource_references: [{ key: { type: ApiResourceType.JOB } }],
            };
            expect(RunUtils.getRecurringRunId(run)).toBe('');
        });
    });

    describe('getRecurringRunName', () => {
        it('returns empty string for undefined run', () => {
            expect(RunUtils.getRecurringRunName(undefined)).toBe('');
        });

        it('returns empty string for run without resource_references', () => {
            const run: ApiRun = {};
            expect(RunUtils.getRecurringRunName(run)).toBe('');
        });

        it('returns empty string for run with no JOB reference', () => {
            const run: ApiRun = {
                resource_references: [{ key: { type: ApiResourceType.EXPERIMENT, id: 'exp-id' } }],
            };
            expect(RunUtils.getRecurringRunName(run)).toBe('');
        });

        it('returns recurring run name from JOB reference', () => {
            const run: ApiRun = {
                resource_references: [
                    { key: { type: ApiResourceType.JOB, id: 'job-id' }, name: 'my-job' },
                ],
            };
            expect(RunUtils.getRecurringRunName(run)).toBe('my-job');
        });

        it('returns empty string for JOB reference without name', () => {
            const run: ApiRun = {
                resource_references: [{ key: { type: ApiResourceType.JOB, id: 'job-id' } }],
            };
            expect(RunUtils.getRecurringRunName(run)).toBe('');
        });
    });
});
