/*
 * Copyright 2022 The Kubeflow Authors
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

import jsyaml from 'js-yaml';
import React from 'react';
import { isSafari } from 'src/lib/Utils';
import Editor from './Editor';

interface PipelineSpecTabContentProps {
  templateString: string;
}

const editorHeightWidth = isSafari() ? '640px' : '100%';

export function PipelineSpecTabContent(props: PipelineSpecTabContentProps) {
  return (
    <Editor
      value={jsyaml.safeDump(jsyaml.safeLoad(props.templateString || ''))}
      // Render the yaml-formatted string in <PipelineSpecTabContent>
      // V1(JSON-formatted):
      //    safeLoad() convert templateString to object first,
      //    safeDump() changes the object to yaml-formatted string
      // V2(YAML-formatted):
      //    Still yaml format after safeLoad() and safeDump().
      height={editorHeightWidth}
      width={editorHeightWidth}
      mode='yaml'
      theme='github'
      editorProps={{ $blockScrolling: true }}
      readOnly={true}
      highlightActiveLine={true}
      showGutter={true}
    />
  );
}
