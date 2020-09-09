// Copyright 2019 Google LLC
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
import fetch from 'node-fetch';
import { GkeMetadataConfigs } from '../configs';

const disabledHandler: Handler = async (_, res) => {
  res.status(500).send('GKE metadata endpoints are disabled.');
};

export const getClusterNameHandler = (options: GkeMetadataConfigs) => {
  if (options.disabled) {
    return disabledHandler;
  }
  return clusterNameHandler;
};

const clusterNameHandler: Handler = async (_, res) => {
  const response = await fetch(
    'http://metadata/computeMetadata/v1/instance/attributes/cluster-name',
    { headers: { 'Metadata-Flavor': 'Google' } },
  );
  if (!response.ok) {
    res.status(500).send('Failed fetching GKE cluster name');
    return;
  }
  res.send(await response.text());
};

export const getProjectIdHandler = (options: GkeMetadataConfigs) => {
  if (options.disabled) {
    return disabledHandler;
  }
  return projectIdHandler;
};

const projectIdHandler: Handler = async (_, res) => {
  const response = await fetch('http://metadata/computeMetadata/v1/project/project-id', {
    headers: { 'Metadata-Flavor': 'Google' },
  });
  if (!response.ok) {
    res.status(500).send('Failed fetching GKE project id');
    return;
  }
  res.send(await response.text());
};
