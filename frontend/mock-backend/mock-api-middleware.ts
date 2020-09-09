// Copyright 2018 Google LLC
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

import * as _path from 'path';
import * as express from 'express';
import * as fs from 'fs';
import RunUtils from '../src/lib/RunUtils';
import helloWorldRuntime from './integration-test-runtime';
import proxyMiddleware from './proxy-middleware';
import { ApiFilter, PredicateOp } from '../src/apis/filter';
import { ApiListExperimentsResponse, ApiExperiment } from '../src/apis/experiment';
import { ApiListJobsResponse, ApiJob } from '../src/apis/job';
import { ApiListPipelinesResponse, ApiPipeline } from '../src/apis/pipeline';
import { ApiListRunsResponse, ApiResourceType, ApiRun, RunStorageState } from '../src/apis/run';
import { ExperimentSortKeys, PipelineSortKeys, RunSortKeys } from '../src/lib/Apis';
import { Response } from 'express-serve-static-core';
import { data as fixedData, namedPipelines } from './fixed-data';

const rocMetadataJsonPath = './eval-output/metadata.json';
const rocMetadataJsonPath2 = './eval-output/metadata2.json';
const rocDataPath = './eval-output/roc.csv';
const rocDataPath2 = './eval-output/roc2.csv';
const tableDataPath = './eval-output/table.csv';

const confusionMatrixMetadataJsonPath = './model-output/metadata.json';
const confusionMatrixPath = './model-output/confusion_matrix.csv';
const helloWorldHtmlPath = './model-output/hello-world.html';
const helloWorldBigHtmlPath = './model-output/hello-world-big.html';

const v1beta1Prefix = '/apis/v1beta1';

let tensorboardPod = '';

// This is a copy of the BaseResource defined within src/pages/ResourceSelector
interface BaseResource {
  id?: string;
  created_at?: Date;
  description?: string;
  name?: string;
  error?: string;
}

// tslint:disable-next-line:no-default-export
export default (app: express.Application) => {
  app.use((req, _, next) => {
    // tslint:disable-next-line:no-console
    console.info(req.method + ' ' + req.originalUrl);
    next();
  });

  proxyMiddleware(app as any, v1beta1Prefix);

  app.set('json spaces', 2);
  app.use(express.json());

  app.get(v1beta1Prefix + '/healthz', (_, res) => {
    res.header('Content-Type', 'application/json');
    res.send({
      apiServerCommitHash: 'd3c4add0a95e930c70a330466d0923827784eb9a',
      apiServerReady: true,
      buildDate: 'Wed Jan 9 19:40:24 UTC 2019',
      frontendCommitHash: '8efb2fcff9f666ba5b101647e909dc9c6889cecb',
    });
  });

  app.get('/hub/', (_, res) => {
    res.sendStatus(200);
  });

  function getSortKeyAndOrder(
    defaultSortKey: string,
    queryParam?: string,
  ): { desc: boolean; key: string } {
    let key = defaultSortKey;
    let desc = false;

    if (queryParam) {
      const keyParts = queryParam.split(' ');
      key = keyParts[0];

      // Check that the key is properly formatted.
      if (
        keyParts.length > 2 ||
        (keyParts.length === 2 && keyParts[1] !== 'asc' && keyParts[1] !== 'desc')
      ) {
        throw new Error(`Invalid sort string: ${queryParam}`);
      }

      desc = keyParts.length === 2 && keyParts[1] === 'desc';
    }
    return { desc, key };
  }

  app.get(v1beta1Prefix + '/jobs', (req, res) => {
    res.header('Content-Type', 'application/json');
    // Note: the way that we use the next_page_token here may not reflect the way the backend works.
    const response: ApiListJobsResponse = {
      jobs: [],
      next_page_token: '',
    };

    let jobs: ApiJob[] = fixedData.jobs;
    if (req.query.filter) {
      jobs = filterResources(fixedData.jobs, req.query.filter);
    }

    const { desc, key } = getSortKeyAndOrder(ExperimentSortKeys.CREATED_AT, req.query.sort_by);

    jobs.sort((a, b) => {
      let result = 1;
      if (a[key]! < b[key]!) {
        result = -1;
      }
      if (a[key]! === b[key]!) {
        result = 0;
      }
      return result * (desc ? -1 : 1);
    });

    const start = req.query.page_token ? +req.query.page_token : 0;
    const end = start + (+req.query.page_size || 20);
    response.jobs = jobs.slice(start, end);

    if (end < jobs.length) {
      response.next_page_token = end + '';
    }

    res.json(response);
  });

  app.get(v1beta1Prefix + '/experiments', (req, res) => {
    res.header('Content-Type', 'application/json');
    // Note: the way that we use the next_page_token here may not reflect the way the backend works.
    const response: ApiListExperimentsResponse = {
      experiments: [],
      next_page_token: '',
    };

    let experiments: ApiExperiment[] = fixedData.experiments;
    if (req.query.filter) {
      experiments = filterResources(fixedData.experiments, req.query.filter);
    }

    const { desc, key } = getSortKeyAndOrder(ExperimentSortKeys.NAME, req.query.sortBy);

    experiments.sort((a, b) => {
      let result = 1;
      if (a[key]! < b[key]!) {
        result = -1;
      }
      if (a[key]! === b[key]!) {
        result = 0;
      }
      return result * (desc ? -1 : 1);
    });

    const start = req.query.pageToken ? +req.query.pageToken : 0;
    const end = start + (+req.query.pageSize || 20);
    response.experiments = experiments.slice(start, end);

    if (end < experiments.length) {
      response.next_page_token = end + '';
    }

    res.json(response);
  });

  app.post(v1beta1Prefix + '/experiments', (req, res) => {
    const experiment: ApiExperiment = req.body;
    if (fixedData.experiments.find(e => e.name!.toLowerCase() === experiment.name!.toLowerCase())) {
      res.status(404).send('An experiment with the same name already exists');
      return;
    }
    experiment.id = 'new-experiment-' + (fixedData.experiments.length + 1);
    fixedData.experiments.push(experiment);

    setTimeout(() => {
      res.send(fixedData.experiments[fixedData.experiments.length - 1]);
    }, 1000);
  });

  app.get(v1beta1Prefix + '/experiments/:eid', (req, res) => {
    res.header('Content-Type', 'application/json');
    const experiment = fixedData.experiments.find(exp => exp.id === req.params.eid);
    if (!experiment) {
      res.status(404).send(`No experiment was found with ID: ${req.params.eid}`);
      return;
    }
    res.json(experiment);
  });

  app.post(v1beta1Prefix + '/jobs', (req, res) => {
    const job: ApiJob = req.body;
    job.id = 'new-job-' + (fixedData.jobs.length + 1);
    job.created_at = new Date();
    job.updated_at = new Date();
    job.enabled = !!job.trigger;
    fixedData.jobs.push(job);

    const date = new Date().toISOString();
    fixedData.runs.push({
      pipeline_runtime: {
        workflow_manifest: JSON.stringify(helloWorldRuntime),
      },
      run: {
        created_at: new Date(),
        id: 'job-at-' + date,
        name: 'job-' + job.name + date,
        scheduled_at: new Date(),
        status: 'Running',
      },
    });
    setTimeout(() => {
      res.send(fixedData.jobs[fixedData.jobs.length - 1]);
    }, 1000);
  });

  app.all(v1beta1Prefix + '/jobs/:jid', (req, res) => {
    res.header('Content-Type', 'application/json');
    switch (req.method) {
      case 'DELETE':
        const i = fixedData.jobs.findIndex(j => j.id === req.params.jid);
        if (fixedData.jobs[i].name!.startsWith('Cannot be deleted')) {
          res.status(502).send(`Deletion failed for job: '${fixedData.jobs[i].name}'`);
        } else {
          // Delete the job from fixedData.
          fixedData.jobs.splice(i, 1);
          res.json({});
        }
        break;
      case 'GET':
        const job = fixedData.jobs.find(j => j.id === req.params.jid);
        if (job) {
          res.json(job);
        } else {
          res.status(404).send(`No job was found with ID: ${req.params.jid}`);
        }
        break;
      default:
        res.status(405).send('Unsupported request type: ' + req.method);
    }
  });

  app.get(v1beta1Prefix + '/runs', (req, res) => {
    res.header('Content-Type', 'application/json');
    // Note: the way that we use the next_page_token here may not reflect the way the backend works.
    const response: ApiListRunsResponse = {
      next_page_token: '',
      runs: [],
    };

    let runs: ApiRun[] = fixedData.runs.map(r => r.run!);

    if (req.query.filter) {
      runs = filterResources(runs, req.query.filter);
    }

    if (req.query['resource_reference_key.type'] === ApiResourceType.EXPERIMENT) {
      runs = runs.filter(r =>
        RunUtils.getAllExperimentReferences(r).some(
          ref =>
            (ref.key && ref.key.id && ref.key.id === req.query['resource_reference_key.id']) ||
            false,
        ),
      );
    }

    const { desc, key } = getSortKeyAndOrder(RunSortKeys.CREATED_AT, req.query.sort_by);

    runs.sort((a, b) => {
      let result = 1;
      if (a[key]! < b[key]!) {
        result = -1;
      }
      if (a[key]! === b[key]!) {
        result = 0;
      }
      return result * (desc ? -1 : 1);
    });

    const start = req.query.page_token ? +req.query.page_token : 0;
    const end = start + (+req.query.page_size || 20);
    response.runs = runs.slice(start, end);

    if (end < runs.length) {
      response.next_page_token = end + '';
    }

    res.json(response);
  });

  app.get(v1beta1Prefix + '/runs/:rid', (req, res) => {
    const rid = req.params.rid;
    const run = fixedData.runs.find(r => r.run!.id === rid);
    if (!run) {
      res.status(404).send('Cannot find a run with id: ' + rid);
      return;
    }
    res.json(run);
  });

  app.post(v1beta1Prefix + '/runs', (req, res) => {
    const date = new Date();
    const run: ApiRun = req.body;
    run.id = 'new-run-' + (fixedData.runs.length + 1);
    run.created_at = date;
    run.scheduled_at = date;
    run.status = 'Running';

    fixedData.runs.push({
      pipeline_runtime: {
        workflow_manifest: JSON.stringify(helloWorldRuntime),
      },
      run,
    });
    setTimeout(() => {
      res.send(fixedData.jobs[fixedData.jobs.length - 1]);
    }, 1000);
  });

  app.post(v1beta1Prefix + '/runs/:rid::method', (req, res) => {
    if (req.params.method !== 'archive' && req.params.method !== 'unarchive') {
      res.status(500).send('Bad method');
    }
    const runDetail = fixedData.runs.find(r => r.run!.id === req.params.rid);
    if (runDetail) {
      runDetail.run!.storage_state =
        req.params.method === 'archive' ? RunStorageState.ARCHIVED : RunStorageState.AVAILABLE;
      res.json({});
    } else {
      res.status(500).send('Cannot find a run with id ' + req.params.rid);
    }
  });

  app.post(v1beta1Prefix + '/jobs/:jid/enable', (req, res) => {
    setTimeout(() => {
      const job = fixedData.jobs.find(j => j.id === req.params.jid);
      if (job) {
        job.enabled = true;
        res.json({});
      } else {
        res.status(500).send('Cannot find a job with id ' + req.params.jid);
      }
    }, 1000);
  });

  app.post(v1beta1Prefix + '/jobs/:jid/disable', (req, res) => {
    setTimeout(() => {
      const job = fixedData.jobs.find(j => j.id === req.params.jid);
      if (job) {
        job.enabled = false;
        res.json({});
      } else {
        res.status(500).send('Cannot find a job with id ' + req.params.jid);
      }
    }, 1000);
  });

  function filterResources(resources: BaseResource[], filterString?: string): BaseResource[] {
    if (!filterString) {
      return resources;
    }
    const filter: ApiFilter = JSON.parse(decodeURIComponent(filterString));
    ((filter && filter.predicates) || []).forEach(p => {
      resources = resources.filter(r => {
        switch (p.op) {
          case PredicateOp.EQUALS:
            if (p.key === 'name') {
              return (
                r.name && r.name.toLocaleLowerCase() === (p.string_value || '').toLocaleLowerCase()
              );
            } else if (p.key === 'storage_state') {
              return (
                (r as ApiRun).storage_state &&
                (r as ApiRun).storage_state!.toString() === p.string_value
              );
            } else {
              throw new Error(`Key: ${p.key} is not yet supported by the mock API server`);
            }
          case PredicateOp.NOTEQUALS:
            if (p.key === 'name') {
              return (
                r.name && r.name.toLocaleLowerCase() !== (p.string_value || '').toLocaleLowerCase()
              );
            } else if (p.key === 'storage_state') {
              return ((r as ApiRun).storage_state || {}).toString() !== p.string_value;
            } else {
              throw new Error(`Key: ${p.key} is not yet supported by the mock API server`);
            }
          case PredicateOp.ISSUBSTRING:
            if (p.key !== 'name') {
              throw new Error(`Key: ${p.key} is not yet supported by the mock API server`);
            }
            return (
              r.name &&
              r.name.toLocaleLowerCase().includes((p.string_value || '').toLocaleLowerCase())
            );
          case PredicateOp.NOTEQUALS:
          // Fall through
          case PredicateOp.GREATERTHAN:
          // Fall through
          case PredicateOp.GREATERTHANEQUALS:
          // Fall through
          case PredicateOp.LESSTHAN:
          // Fall through
          case PredicateOp.LESSTHANEQUALS:
            // Fall through
            throw new Error(`Op: ${p.op} is not yet supported by the mock API server`);
          default:
            throw new Error(`Unknown Predicate op: ${p.op}`);
        }
      });
    });
    return resources;
  }

  app.get(v1beta1Prefix + '/pipelines', (req, res) => {
    res.header('Content-Type', 'application/json');
    const response: ApiListPipelinesResponse = {
      next_page_token: '',
      pipelines: [],
    };

    let pipelines: ApiPipeline[] = fixedData.pipelines;
    if (req.query.filter) {
      pipelines = filterResources(fixedData.pipelines, req.query.filter);
    }

    const { desc, key } = getSortKeyAndOrder(PipelineSortKeys.CREATED_AT, req.query.sort_by);

    pipelines.sort((a, b) => {
      let result = 1;
      if (a[key]! < b[key]!) {
        result = -1;
      }
      if (a[key]! === b[key]!) {
        result = 0;
      }
      return result * (desc ? -1 : 1);
    });

    const start = req.query.page_token ? +req.query.page_token : 0;
    const end = start + (+req.query.page_size || 20);
    response.pipelines = pipelines.slice(start, end);

    if (end < pipelines.length) {
      response.next_page_token = end + '';
    }

    res.json(response);
  });

  app.delete(v1beta1Prefix + '/pipelines/:pid', (req, res) => {
    res.header('Content-Type', 'application/json');
    const i = fixedData.pipelines.findIndex(p => p.id === req.params.pid);

    if (i === -1) {
      res.status(404).send(`No pipelines was found with ID: ${req.params.pid}`);
      return;
    }

    if (fixedData.pipelines[i].name!.startsWith('Cannot be deleted')) {
      res.status(502).send(`Deletion failed for pipeline: '${fixedData.pipelines[i].name}'`);
      return;
    }
    // Delete the pipelines from fixedData.
    fixedData.pipelines.splice(i, 1);
    res.json({});
  });

  app.get(v1beta1Prefix + '/pipelines/:pid', (req, res) => {
    res.header('Content-Type', 'application/json');
    const pipeline = fixedData.pipelines.find(p => p.id === req.params.pid);
    if (!pipeline) {
      res.status(404).send(`No pipeline was found with ID: ${req.params.pid}`);
      return;
    }
    res.json(pipeline);
  });

  app.get(v1beta1Prefix + '/pipelines/:pid/templates', (req, res) => {
    res.header('Content-Type', 'text/x-yaml');
    const pipeline = fixedData.pipelines.find(p => p.id === req.params.pid);
    if (!pipeline) {
      res.status(404).send(`No pipeline was found with ID: ${req.params.pid}`);
      return;
    }
    let filePath = '';
    if (req.params.pid === namedPipelines.noParamsPipeline.id) {
      filePath = './mock-backend/mock-conditional-template.yaml';
    } else if (req.params.pid === namedPipelines.unstructuredTextPipeline.id) {
      filePath = './mock-backend/mock-recursive-template.yaml';
    } else {
      filePath = './mock-backend/mock-template.yaml';
    }
    res.send(JSON.stringify({ template: fs.readFileSync(filePath, 'utf-8') }));
  });

  function mockCreatePipeline(res: Response, name: string, body?: any): void {
    res.header('Content-Type', 'application/json');
    // Don't allow uploading multiple pipelines with the same name
    if (fixedData.pipelines.find(p => p.name === name)) {
      res
        .status(502)
        .send(`A Pipeline named: "${name}" already exists. Please choose a different name.`);
    } else {
      const pipeline = body || {};
      pipeline.id = 'new-pipeline-' + (fixedData.pipelines.length + 1);
      pipeline.name = name;
      pipeline.created_at = new Date();
      pipeline.description =
        'TODO: the mock middleware does not actually use the uploaded pipeline';
      pipeline.parameters = [
        {
          name: 'output',
        },
        {
          name: 'param-1',
        },
        {
          name: 'param-2',
        },
      ];
      fixedData.pipelines.push(pipeline);
      setTimeout(() => {
        res.send(fixedData.pipelines[fixedData.pipelines.length - 1]);
      }, 1000);
    }
  }

  app.post(v1beta1Prefix + '/pipelines', (req, res) => {
    mockCreatePipeline(res, req.body.name);
  });

  app.post(v1beta1Prefix + '/pipelines/upload', (req, res) => {
    mockCreatePipeline(res, decodeURIComponent(req.query.name), req.body);
  });

  app.get('/artifacts/get', (req, res) => {
    const key = decodeURIComponent(req.query.key);
    res.header('Content-Type', 'application/json');
    if (key.endsWith('roc.csv')) {
      res.sendFile(_path.resolve(__dirname, rocDataPath));
    } else if (key.endsWith('roc2.csv')) {
      res.sendFile(_path.resolve(__dirname, rocDataPath2));
    } else if (key.endsWith('confusion_matrix.csv')) {
      res.sendFile(_path.resolve(__dirname, confusionMatrixPath));
    } else if (key.endsWith('table.csv')) {
      res.sendFile(_path.resolve(__dirname, tableDataPath));
    } else if (key.endsWith('hello-world.html')) {
      res.sendFile(_path.resolve(__dirname, helloWorldHtmlPath));
    } else if (key.endsWith('hello-world-big.html')) {
      res.sendFile(_path.resolve(__dirname, helloWorldBigHtmlPath));
    } else if (key === 'analysis') {
      res.sendFile(_path.resolve(__dirname, confusionMatrixMetadataJsonPath));
    } else if (key === 'analysis2') {
      res.sendFile(_path.resolve(__dirname, confusionMatrixMetadataJsonPath));
    } else if (key === 'model') {
      res.sendFile(_path.resolve(__dirname, rocMetadataJsonPath));
    } else if (key === 'model2') {
      res.sendFile(_path.resolve(__dirname, rocMetadataJsonPath2));
    } else {
      // TODO: what does production return here?
      res.send('dummy file for key: ' + key);
    }
  });

  app.get('/apps/tensorboard', (req, res) => {
    res.send(tensorboardPod);
  });

  app.post('/apps/tensorboard', (req, res) => {
    tensorboardPod = 'http://tensorboardserver:port';
    setTimeout(() => {
      res.send('ok');
    }, 1000);
  });

  app.get('/k8s/pod/logs', (req, res) => {
    const podName = decodeURIComponent(req.query.podname);
    if (podName === 'json-12abc') {
      res.status(404).send('pod not found');
      return;
    }
    if (podName === 'coinflip-recursive-q7dqb-3721646052') {
      res.status(500).send('Failed to retrieve log');
      return;
    }
    const shortLog = fs.readFileSync('./mock-backend/shortlog.txt', 'utf-8');
    const longLog = fs.readFileSync('./mock-backend/longlog.txt', 'utf-8');
    const log = podName === 'coinflip-recursive-q7dqb-3466727817' ? longLog : shortLog;
    setTimeout(() => {
      res.send(log);
    }, 300);
  });

  app.get('/visualizations/allowed', (req, res) => {
    res.send(true);
  });

  // Uncomment this instead to test 404 endpoints.
  // app.get('/system/cluster-name', (_, res) => {
  //   res.status(404).send('404 Not Found');
  // });
  // app.get('/system/project-id', (_, res) => {
  //   res.status(404).send('404 Not Found');
  // });
  app.get('/system/cluster-name', (_, res) => {
    res.send('mock-cluster-name');
  });
  app.get('/system/project-id', (_, res) => {
    res.send('mock-project-id');
  });

  app.all(v1beta1Prefix + '*', (req, res) => {
    res.status(404).send('Bad request endpoint.');
  });
};
