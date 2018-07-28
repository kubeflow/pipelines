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

export default (app) => {

  app.set('json spaces', 2);
  app.use(express.json());

  app.get(v1alpha2Prefix + '/healthz', (req, res) => {
    if (apiServerReady) {
      res.send({ apiServerReady });
    } else {
      res.status(404).send();
    }
  });

  app.get(v1alpha2Prefix + '/pipelines', (req, res) => {
    if (!apiServerReady) {
      res.status(404).send();
      return;
    }

    res.header('Content-Type', 'application/json');
    // Note: the way that we use the next_page_token here may not reflect the way the backend works.
    const response: ListPipelinesResponse = {
      next_page_token: '',
      pipelines: [],
    };

    let pipelines: Pipeline[] = fixedData.pipelines;
    if (req.query.filterBy) {
      // NOTE: We do not mock fuzzy matching. E.g. 'ee' doesn't match 'Pipeline'
      // This may need to be updated when the backend implements filtering.
      pipelines = fixedData.pipelines.filter((p) =>
          p.name.toLocaleLowerCase().indexOf(
              decodeURIComponent(req.query.filterBy).toLocaleLowerCase()) > -1);

    }

    // The backend sorts by created_at by default.
    const sortKey = req.query.sortBy || PipelineSortKeys.CREATED_AT;

    if (!isValidSortKey(PipelineSortKeys, sortKey)) {
      res.status(405).send(`Unsupported sort string: ${sortKey}`);
      return;
    }

    pipelines.sort((a, b) => {
      let result = 1;
      if (a[sortKey] < b[sortKey]) {
        result = -1;
      }
      if (a[sortKey] === b[sortKey]) {
        result = 0;
      }
      return result * ((req.query.ascending === 'true') ? 1 : -1);
    });

    const start = (req.query.pageToken ? +req.query.pageToken : 0);
    const end = start + (+req.query.pageSize);
    response.pipelines = pipelines.slice(start, end);

    if (end < pipelines.length) {
      response.next_page_token = end + '';
    }

    res.json(response);
  });

  app.post(v1alpha2Prefix + '/pipelines', (req, res) => {
    const pipeline = req.body;
    pipeline.id = (fixedData.pipelines.length + 1) + '';
    pipeline.created_at = new Date().toISOString();
    pipeline.updated_at = new Date().toISOString();
    pipeline.jobs = [fixedData.jobs[0]];
    pipeline.enabled = !!pipeline.trigger;
    fixedData.pipelines.push(pipeline);
    setTimeout(() => {
      res.send(fixedData.pipelines[0]);
    }, 1000);
  });

  app.all(v1alpha2Prefix + '/pipelines/:pid', (req, res) => {
    res.header('Content-Type', 'application/json');
    switch (req.method) {
      case 'DELETE':
        const i = fixedData.pipelines.findIndex((p) => p.id === req.params.pid);
        if (fixedData.pipelines[i].name.startsWith('Cannot be deleted')) {
          res.status(502).send(`Deletion failed for Pipeline: '${fixedData.pipelines[i].name}'`);
        } else {
          // Delete the Pipeline from fixedData.
          fixedData.pipelines.splice(i, 1);
          res.send('ok');
        }
        break;
      case 'GET':
        res.json(fixedData.pipelines.find((p) => p.id === req.params.pid));
        break;
      default:
        res.status(405).send('Unsupported request type: ' + res.method);
    }
  });

  app.get(v1alpha2Prefix + '/pipelines/:pid/jobs', (req, res) => {
    res.header('Content-Type', 'application/json');
    // Note: the way that we use the next_page_token here may not reflect the way the backend works.
    const response: ListJobsResponse = {
      jobs: [],
      next_page_token: '',
    };

    let jobs: JobMetadata[] =
        fixedData.pipelines.find((p) => p.id === req.params.pid).jobs.map((j) => j.job);

    if (req.query.filterBy) {
      // NOTE: We do not mock fuzzy matching. E.g. 'ee' doesn't match 'Pipeline'
      // This may need to be updated when the backend implements filtering.
      jobs = jobs.filter((j) => j.name.toLocaleLowerCase().indexOf(
          decodeURIComponent(req.query.filterBy).toLocaleLowerCase()) > -1);
    }

    // The backend sorts by created_at by default.
    const sortKey = req.query.sortBy || JobSortKeys.CREATED_AT;

    if (!isValidSortKey(JobSortKeys, sortKey)) {
      res.status(405).send(`Unsupported sort string: ${sortKey}`);
      return;
    }

    jobs.sort((a, b) => {
      let result = 1;
      if (a[sortKey] < b[sortKey]) {
        result = -1;
      }
      if (a[sortKey] === b[sortKey]) {
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

  app.post(v1alpha2Prefix + '/pipelines/:pid/enable', (req, res) => {
    setTimeout(() => {
      const pipeline = fixedData.pipelines.find((p) => p.id === req.params.pid);
      pipeline.enabled = true;
      res.send('ok');
    }, 1000);
  });

  app.post(v1alpha2Prefix + '/pipelines/:pid/disable', (req, res) => {
    setTimeout(() => {
      const pipeline = fixedData.pipelines.find((p) => p.id === req.params.pid);
      pipeline.enabled = false;
      res.send('ok');
    }, 1000);
  });

  app.get(v1alpha2Prefix + '/pipelines/:pid/jobs/:jid', (req, res) => {
    const jid = req.params.jid;
    const pipeline = fixedData.pipelines.find((p) => p.id === req.params.pid);
    const job = pipeline.jobs.find((j) => j.job.id === jid);
    if (!job) {
      res.status(404).send('Cannot find a job with id: ' + jid);
      return;
    }
    res.json(job);
  });

  app.get(v1alpha2Prefix + '/packages', (req, res) => {
    res.header('Content-Type', 'application/json');
    const response: ListPackagesResponse = {
      next_page_token: '',
      packages: fixedData.packages,
    };
    res.json(response);
  });

  app.get(v1alpha2Prefix + '/packages/:pid/templates', (req, res) => {
    res.header('Content-Type', 'text/x-yaml');
    res.send(JSON.stringify(
      { template: fs.readFileSync('./mock-backend/mock-template.yaml', 'utf-8') }));
  });

  app.post(v1alpha2Prefix + '/packages/upload', (req, res) => {
    res.header('Content-Type', 'application/json');
    res.json(fixedData.packages[0]);
  });

  app.get('/artifacts/list/:path', (req, res) => {

    const path = decodeURIComponent(req.params.path);

    res.header('Content-Type', 'application/json');
    res.json([
      path + '/file1',
      path + '/file2',
      path + (path.match('analysis$|model$') ? '/metadata.json' : '/file3'),
    ]);
  });

  app.get('/artifacts/get/:path', (req, res) => {
    res.header('Content-Type', 'application/json');
    const path = decodeURIComponent(req.params.path);
    if (path.endsWith('roc.csv')) {
      res.sendFile(_path.resolve(__dirname, rocDataPath));
    } else if (path.endsWith('confusion_matrix.csv')) {
      res.sendFile(_path.resolve(__dirname, confusionMatrixPath));
    } else if (path.endsWith('table.csv')) {
      res.sendFile(_path.resolve(__dirname, tableDataPath));
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

  proxyMiddleware(app, v1alpha2Prefix);

};
