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

import express from 'express';
import mockApiMiddleware from './mock-api-middleware';

const app = express();
const port = process.argv[2] || 3001;

// Uncomment the following line to get 1000ms delay to all requests
// app.use((req, res, next) => { setTimeout(next, 1000); });

app.use((_: any, res: any, next: any) => {
  res.header('Access-Control-Allow-Origin', '*');
  res.header('Access-Control-Allow-Headers', 'X-Requested-With, content-type');
  res.header('Access-Control-Allow-Methods', 'GET, POST, OPTIONS, PUT, DELETE');
  next();
});

mockApiMiddleware(app as any);

app.listen(port, () => {
  // tslint:disable-next-line:no-console
  console.log('Server listening at http://localhost:' + port);
});
