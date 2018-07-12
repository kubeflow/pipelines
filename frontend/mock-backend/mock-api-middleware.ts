import * as express from 'express';
import * as fs from 'fs';
import * as _path from 'path';
import proxyMiddleware from '../server/proxy-middleware';

import { JobMetadata } from '../src/api/job';
import { JobSortKeys } from '../src/api/list_jobs_request';
import { ListJobsResponse } from '../src/api/list_jobs_response';
import { ListPackagesResponse } from '../src/api/list_packages_response';
import { PipelineSortKeys } from '../src/api/list_pipelines_request';
import { ListPipelinesResponse } from '../src/api/list_pipelines_response';
import { Pipeline } from '../src/api/pipeline';

const prefix = __dirname + '/pipeline-data';

const fixedData = require('./fixed-data').data;

const rocMetadataJsonPath = './eval-output/metadata.json';
const rocDataPath = './eval-output/roc.csv';

const confusionMatrixMetadataJsonPath = './model-output/metadata.json';
const confusionMatrixPath = './model-output/confusion_matrix.csv';
const staticHtmlPath = './model-output/hello-world.html';

const apisPrefix = '/apis/v1alpha1';

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

export default (app) => {

  app.set('json spaces', 2);
  app.use(express.json());

  app.get(apisPrefix + '/healthz', (req, res) => {
    if (apiServerReady) {
      res.send({ apiServerReady });
    } else {
      res.status(404).send();
    }
  });

  app.get(apisPrefix + '/pipelines', (req, res) => {
    if (!apiServerReady) {
      res.status(404).send();
      return;
    }

    res.header('Content-Type', 'application/json');
    // Note: the way that we use the nextPageToken here may not reflect the way the backend works.
    const response: ListPipelinesResponse = {
      nextPageToken: '',
      pipelines: [],
    };

    let pipelines: Pipeline[] = fixedData.pipelines;
    if (req.query.filterBy) {
      // NOTE: We do not mock fuzzy matching. E.g. 'ee' doesn't match 'Pipeline'
      // This may need to be updated when the backend implements filtering.
      pipelines = fixedData.pipelines.filter((p) => p.name.toLocaleLowerCase().match(
          decodeURIComponent(req.query.filterBy).toLocaleLowerCase()));

    }

    if (req.query.sortBy) {
      if (!isValidSortKey(PipelineSortKeys, req.query.sortBy)) {
        res.status(405).send(`Unsupported sort string: ${req.query.sortBy}`);
        return;
      }
      pipelines.sort((a, b) => {
        let result = 1;
        if (a[req.query.sortBy] < b[req.query.sortBy]) {
          result = -1;
        }
        if (a[req.query.sortBy] === b[req.query.sortBy]) {
          result = 0;
        }
        return result * ((req.query.ascending === 'true') ? 1 : -1);
      });
    }

    const start = (req.query.pageToken ? +req.query.pageToken : 0);
    const end = start + (+req.query.pageSize);
    response.pipelines = pipelines.slice(start, end);

    if (end < pipelines.length) {
      response.nextPageToken = end + '';
    }

    res.json(response);
  });

  app.post(apisPrefix + '/pipelines', (req, res) => {
    const pipeline = req.body;
    pipeline.id = fixedData.pipelines.length + 1;
    pipeline.created_at = new Date().toISOString();
    pipeline.jobs = [fixedData.jobs[0]];
    pipeline.enabled = !!pipeline.schedule;
    fixedData.pipelines.push(pipeline);
    setTimeout(() => {
      res.send(fixedData.pipelines[0]);
    }, 1000);
  });

  app.all(apisPrefix + '/pipelines/:pid', (req, res) => {
    res.header('Content-Type', 'application/json');
    const pid = Number.parseInt(req.params.pid);
    switch (req.method) {
      case 'DELETE':
        const i = fixedData.pipelines.findIndex((p) => p.id === pid);
        if (fixedData.pipelines[i].name.startsWith('Cannot be deleted')) {
          res.status(502).send(`Deletion failed for Pipeline: '${fixedData.pipelines[i].name}'`);
        } else {
          // Delete the Pipeline from fixedData.
          fixedData.pipelines.splice(i, 1);
          res.send('ok');
        }
        break;
      case 'GET':
        res.json(fixedData.pipelines.find((p) => p.id === pid));
        break;
      default:
        res.status(405).send('Unsupported request type: ' + res.method);
    }
  });

  app.get(apisPrefix + '/pipelines/:pid/jobs', (req, res) => {
    res.header('Content-Type', 'application/json');
    const pid = Number.parseInt(req.params.pid);
    // Note: the way that we use the nextPageToken here may not reflect the way the backend works.
    const response: ListJobsResponse = {
      jobs: [],
      nextPageToken: '',
    };

    let jobs: JobMetadata[] =
        fixedData.pipelines.find((p) => p.id === pid).jobs.map((j) => j.job);

    if (req.query.filterBy) {
      // NOTE: We do not mock fuzzy matching. E.g. 'ee' doesn't match 'Pipeline'
      // This may need to be updated when the backend implements filtering.
      jobs = jobs.filter((j) => j.name.toLocaleLowerCase().match(
          decodeURIComponent(req.query.filterBy).toLocaleLowerCase()));
    }

    // The backend sorts by created_at by default.
    req.query.sortBy = req.query.sortBy || JobSortKeys.CREATED_AT;

    if (req.query.sortBy) {
      if (!isValidSortKey(JobSortKeys, req.query.sortBy)) {
        res.status(405).send(`Unsupported sort string: ${req.query.sortBy}`);
        return;
      }
      jobs.sort((a, b) => {
        let result = 1;
        if (a[req.query.sortBy] < b[req.query.sortBy]) {
          result = -1;
        }
        if (a[req.query.sortBy] === b[req.query.sortBy]) {
          result = 0;
        }
        return result * ((req.query.ascending === 'true') ? 1 : -1);
      });
    }

    const start = (req.query.pageToken ? +req.query.pageToken : 0);
    const end = start + (+req.query.pageSize);
    response.jobs = jobs.slice(start, end);

    if (end < jobs.length) {
      response.nextPageToken = end + '';
    }

    res.json(response);
  });

  app.post(apisPrefix + '/pipelines/:pid/enable', (req, res) => {
    setTimeout(() => {
      const pid = Number.parseInt(req.params.pid);
      const pipeline = fixedData.pipelines.find((p) => p.id === pid);
      pipeline.enabled = true;
      res.send('ok');
    }, 1000);
  });

  app.post(apisPrefix + '/pipelines/:pid/disable', (req, res) => {
    setTimeout(() => {
      const pid = Number.parseInt(req.params.pid);
      const pipeline = fixedData.pipelines.find((p) => p.id === pid);
      pipeline.enabled = false;
      res.send('ok');
    }, 1000);
  });

  app.get(apisPrefix + '/pipelines/:pid/jobs/:jname', (req, res) => {
    const pid = Number.parseInt(req.params.pid);
    const jname = req.params.jname;
    const pipeline = fixedData.pipelines.find((p) => p.id === pid);
    const job = pipeline.jobs.find((j) => j.job.name === jname);
    if (!job) {
      res.status(404).send('Cannot find a job with name: ' + jname);
      return;
    }
    res.json(job);
  });

  app.get(apisPrefix + '/packages', (req, res) => {
    res.header('Content-Type', 'application/json');
    const response: ListPackagesResponse = {
      nextPageToken: '',
      packages: fixedData.packages,
    };
    res.json(response);
  });

  app.get(apisPrefix + '/packages/:pid/templates', (req, res) => {
    res.header('Content-Type', 'text/x-yaml');
    res.send(JSON.stringify(
      { template: fs.readFileSync('./mock-backend/mock-template.yaml', 'utf-8') }));
  });

  app.post(apisPrefix + '/packages/upload', (req, res) => {
    res.header('Content-Type', 'application/json');
    res.json(fixedData.packages[0]);
  });

  app.get(apisPrefix + '/artifacts/list/:path', (req, res) => {

    const path = decodeURIComponent(req.params.path);

    res.header('Content-Type', 'application/json');
    res.json([
      path + '/file1',
      path + '/file2',
      path + (path.match('analysis$|model$') ? '/metadata.json' : '/file3'),
    ]);
  });

  app.get(apisPrefix + '/artifacts/get/:path', (req, res) => {
    res.header('Content-Type', 'application/json');
    const path = decodeURIComponent(req.params.path);
    if (path.endsWith('roc.csv')) {
      res.sendFile(_path.resolve(__dirname, rocDataPath));
    } else if (path.endsWith('confusion_matrix.csv')) {
      res.sendFile(_path.resolve(__dirname, confusionMatrixPath));
    } else if (path.endsWith('hello-world.html')) {
      res.sendFile(_path.resolve(__dirname, staticHtmlPath));
    } else if (path.endsWith('analysis/metadata.json')) {
      res.sendFile(_path.resolve(__dirname, confusionMatrixMetadataJsonPath));
    } else if (path.endsWith('model/metadata.json')) {
      res.sendFile(_path.resolve(__dirname, rocMetadataJsonPath));
    } else {
      res.send('dummy file');
    }
  });

  app.get(apisPrefix + '/apps/tensorboard', (req, res) => {
    res.send(tensorboardPod);
  });

  app.post(apisPrefix + '/apps/tensorboard', (req, res) => {
    tensorboardPod = 'http://tensorboardserver:port';
    setTimeout(() => {
      res.send('ok');
    }, 1000);
  });

  app.get(apisPrefix + '/k8s/pod/logs', (req, res) => {
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

  app.all(apisPrefix + '*', (req, res) => {
    res.status(404).send('Bad request endpoint.');
  });

  proxyMiddleware(app, apisPrefix);

};
