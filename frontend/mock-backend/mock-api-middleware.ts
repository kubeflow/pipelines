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

import * as express from 'express';
import * as fs from 'fs';
import * as _path from 'path';
import proxyMiddleware from '../server/proxy-middleware';

import { Job } from '../src/api/job';
import { JobSortKeys } from '../src/api/list_jobs_request';
import { ListJobsResponse } from '../src/api/list_jobs_response';
import { ListPipelinesResponse } from '../src/api/list_pipelines_response';
import { RunSortKeys } from '../src/api/list_runs_request';
import { ListRunsResponse } from '../src/api/list_runs_response';
import { RunMetadata } from '../src/api/run';
import { data as fixedData } from './fixed-data';

const rocMetadataJsonPath = './eval-output/metadata.json';
const rocDataPath = './eval-output/roc.csv';
const tableDataPath = './eval-output/table.csv';

const confusionMatrixMetadataJsonPath = './model-output/metadata.json';
const confusionMatrixPath = './model-output/confusion_matrix.csv';
const staticHtmlPath = './model-output/hello-world.html';

const v1alpha2Prefix = '/apis/v1alpha2';

let tensorboardPod = '';

let apiServerReady = false;

// Simulate API server not ready for 5 seconds
setTimeout(() => {
  apiServerReady = true;
}, 5000);

function isValidSortKey(sortKeyEnumType: any, key: string): boolean {
  const findResult = Object.keys(sortKeyEnumType).find(
      (k) => sortKeyEnumType[k] === key
  );
  return !!findResult;
}

export default (app: express.Application) => {

  app.set('json spaces', 2);
  app.use(express.json());

  app.get(v1alpha2Prefix + '/healthz', (req, res) => {
    if (apiServerReady) {
      res.send({ apiServerReady });
    } else {
      res.status(404).send();
    }
  });

  app.get(v1alpha2Prefix + '/jobs', (req, res) => {
    if (!apiServerReady) {
      res.status(404).send();
      return;
    }

    res.header('Content-Type', 'application/json');
    // Note: the way that we use the next_page_token here may not reflect the way the backend works.
    const response: ListJobsResponse = {
      jobs: [],
      next_page_token: '',
    };

    let jobs: Job[] = fixedData.jobs;
    if (req.query.filterBy) {
      // NOTE: We do not mock fuzzy matching. E.g. 'jb' doesn't match 'job'
      // This may need to be updated when the backend implements filtering.
      jobs = fixedData.jobs.filter((p) =>
          p.name.toLocaleLowerCase().indexOf(
              decodeURIComponent(req.query.filterBy).toLocaleLowerCase()) > -1);

    }

    // The backend sorts by created_at by default.
    const sortKey: (keyof Job) = req.query.sortBy || JobSortKeys.CREATED_AT;

    if (!isValidSortKey(JobSortKeys, sortKey)) {
      res.status(405).send(`Unsupported sort string: ${sortKey}`);
      return;
    }

    jobs.sort((a, b) => {
      let result = 1;
      if (a[sortKey]! < b[sortKey]!) {
        result = -1;
      }
      if (a[sortKey]! === b[sortKey]!) {
        result = 0;
      }
      return result * ((req.query.ascending === 'true') ? 1 : -1);
    });

    const start = (req.query.pageToken ? +req.query.pageToken : 0);
    const end = start + (+req.query.pageSize);
    response.jobs = jobs.slice(start, end);

    if (end < jobs.length) {
      response.next_page_token = end + '';
    }

    res.json(response);
  });

  app.post(v1alpha2Prefix + '/jobs', (req, res) => {
    const job = req.body;
    job.id = 'new-job-' + (fixedData.jobs.length + 1);
    job.created_at = new Date().toISOString();
    job.updated_at = new Date().toISOString();
    job.runs = [fixedData.runs[0]];
    job.enabled = !!job.trigger;
    fixedData.jobs.push(job);
    setTimeout(() => {
      res.send(fixedData.jobs[0]);
    }, 1000);
  });

  app.all(v1alpha2Prefix + '/jobs/:pid', (req, res) => {
    res.header('Content-Type', 'application/json');
    switch (req.method) {
      case 'DELETE':
        const i = fixedData.jobs.findIndex((p) => p.id === req.params.pid);
        if (fixedData.jobs[i].name.startsWith('Cannot be deleted')) {
          res.status(502).send(`Deletion failed for job: '${fixedData.jobs[i].name}'`);
        } else {
          // Delete the job from fixedData.
          fixedData.jobs.splice(i, 1);
          res.send('ok');
        }
        break;
      case 'GET':
        res.json(fixedData.jobs.find((p) => p.id === req.params.pid));
        break;
      default:
        res.status(405).send('Unsupported request type: ' + req.method);
    }
  });

  app.get(v1alpha2Prefix + '/jobs/:jid/runs', (req, res) => {
    res.header('Content-Type', 'application/json');
    // Note: the way that we use the next_page_token here may not reflect the way the backend works.
    const response: ListRunsResponse = {
      next_page_token: '',
      runs: [],
    };

    let runs: RunMetadata[] = fixedData.runs.map((r) => r.run);

    if (req.params.jid.startsWith('new-job-')) {
      response.runs = runs.slice(0, 1);
      res.json(response);
      return;
    } else if (req.params.jid === '7fc01714-4a13-4c05-5902-a8a72c14253b') { // No runs job
      res.json(response);
      return;
    }

    if (req.query.filterBy) {
      // NOTE: We do not mock fuzzy matching. E.g. 'jb' doesn't match 'job'
      // This may need to be updated when the backend implements filtering.
      runs = runs.filter((r) => r.name.toLocaleLowerCase().indexOf(
          decodeURIComponent(req.query.filterBy).toLocaleLowerCase()) > -1);
    }

    // The backend sorts by created_at by default.
    const sortKey: (keyof RunMetadata) = req.query.sortBy || RunSortKeys.CREATED_AT;

    if (!isValidSortKey(RunSortKeys, sortKey)) {
      res.status(405).send(`Unsupported sort string: ${sortKey}`);
      return;
    }

    runs.sort((a, b) => {
      let result = 1;
      if (a[sortKey]! < b[sortKey]!) {
        result = -1;
      }
      if (a[sortKey]! === b[sortKey]!) {
        result = 0;
      }
      return result * ((req.query.ascending === 'true') ? 1 : -1);
    });

    const start = (req.query.pageToken ? +req.query.pageToken : 0);
    const end = start + (+req.query.pageSize);
    response.runs = runs.slice(start, end);

    if (end < runs.length) {
      response.next_page_token = end + '';
    }

    res.json(response);
  });

  app.post(v1alpha2Prefix + '/jobs/:pid/enable', (req, res) => {
    setTimeout(() => {
      const job = fixedData.jobs.find((p) => p.id === req.params.pid);
      if (job) {
        job.enabled = true;
        res.send('ok');
      } else {
        res.status(500).send('Cannot find a job with id ' + req.params.pid);
      }
    }, 1000);
  });

  app.post(v1alpha2Prefix + '/jobs/:pid/disable', (req, res) => {
    setTimeout(() => {
      const job = fixedData.jobs.find((p) => p.id === req.params.pid);
      if (job) {
        job.enabled = false;
        res.send('ok');
      } else {
        res.status(500).send('Cannot find a job with id ' + req.params.pid);
      }
    }, 1000);
  });

  app.get(v1alpha2Prefix + '/jobs/:pid/runs/:jid', (req, res) => {
    const jid = req.params.jid;
    const run = fixedData.runs.find((r) => r.run.id === jid);
    if (!run) {
      res.status(404).send('Cannot find a run with id: ' + jid);
      return;
    }
    res.json(run);
  });

  app.get(v1alpha2Prefix + '/pipelines', (req, res) => {
    res.header('Content-Type', 'application/json');
    const response: ListPipelinesResponse = {
      next_page_token: '',
      pipelines: fixedData.pipelines,
    };
    res.json(response);
  });

  app.get(v1alpha2Prefix + '/pipelines/:pid', (req, res) => {
    const pid = req.params.pid;
    const pipeline = fixedData.pipelines.find((p) => p.id === pid);
    if (!pipeline) {
      res.status(404).send('Cannot find a pipeline with id: ' + pid);
      return;
    }
    res.json(pipeline);
  });

  app.get(v1alpha2Prefix + '/pipelines/:pid/templates', (req, res) => {
    res.header('Content-Type', 'text/x-yaml');
    res.send(JSON.stringify(
      { template: fs.readFileSync('./mock-backend/mock-template.yaml', 'utf-8') }));
  });

  app.post(v1alpha2Prefix + '/pipelines/upload', (req, res) => {
    res.header('Content-Type', 'application/json');
    res.json(fixedData.pipelines[0]);
  });

  app.get('/artifacts/get', (req, res) => {
    const key = decodeURIComponent(req.query.key);
    res.header('Content-Type', 'application/json');
    if (key.endsWith('roc.csv')) {
      res.sendFile(_path.resolve(__dirname, rocDataPath));
    } else if (key.endsWith('confusion_matrix.csv')) {
      res.sendFile(_path.resolve(__dirname, confusionMatrixPath));
    } else if (key.endsWith('table.csv')) {
      res.sendFile(_path.resolve(__dirname, tableDataPath));
    } else if (key.endsWith('hello-world.html')) {
      res.sendFile(_path.resolve(__dirname, staticHtmlPath));
    } else if (key === 'analysis') {
      res.sendFile(_path.resolve(__dirname, confusionMatrixMetadataJsonPath));
    } else if (key === 'model') {
      res.sendFile(_path.resolve(__dirname, rocMetadataJsonPath));
    } else {
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
    setTimeout(() => {
      res.send(String.raw`
      _____________
    < hello world >
      -------------
           \
            \
             \
                           ##        .
                     ## ## ##       ==
                  ## ## ## ##      ===
              /""""""""""""""""___/ ===
         ~~~ {~~ ~~~~ ~~~ ~~~~ ~~ ~ /  ===- ~~~
              \______ o          __/
               \    \        __/
                 \____\______/

       `);
    }, 300);
  });

  app.get('/_componentTests*', (req, res) => {
    res.sendFile(_path.resolve('test', 'components', 'index.html'));
  });

  app.all(v1alpha2Prefix + '*', (req, res) => {
    res.status(404).send('Bad request endpoint.');
  });

  proxyMiddleware(app as any, v1alpha2Prefix);

};
