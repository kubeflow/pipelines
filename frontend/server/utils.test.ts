// Copyright 2020 Google LLC
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
import { PassThrough } from 'stream';
import { PreviewStream } from './utils';

describe('utils', () => {
  describe('PreviewStream', () => {
    it('should stream first 5 bytes', done => {
      const peek = 5;
      const input = 'some string that will be truncated.';
      const source = new PassThrough();
      const preview = new PreviewStream({ peek });
      const dst = source.pipe(preview).on('end', done);
      source.end(input);
      dst.once('readable', () => expect(dst.read().toString()).toBe(input.slice(0, peek)));
    });

    it('should stream everything if peek==0', done => {
      const peek = 0;
      const input = 'some string that will be truncated.';
      const source = new PassThrough();
      const preview = new PreviewStream({ peek });
      const dst = source.pipe(preview).on('end', done);
      source.end(input);
      dst.once('readable', () => expect(dst.read().toString()).toBe(input));
    });
  });
});
