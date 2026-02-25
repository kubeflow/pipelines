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

import React, { useState } from 'react';
import MD2Tabs from 'src/atoms/MD2Tabs';
import { commonCss, padding } from 'src/Css';
import { classes } from 'typestyle';
import { PageProps } from 'src/pages/Page';
import ArtifactListView from './ArtifactList';

function ArtifactListSwitcher(props: PageProps) {
  const [selectedTab, setSelectedTab] = useState(0);

  return (
    <div className={classes(commonCss.page, padding(20, 't'))}>
      <MD2Tabs tabs={['Default', 'Grouped']} selectedTab={selectedTab} onSwitch={setSelectedTab} />

      {selectedTab === 0 && <ArtifactListView {...props} isGroupView={false} />}

      {selectedTab === 1 && <ArtifactListView {...props} isGroupView={true} />}
    </div>
  );
}

export default ArtifactListSwitcher;
