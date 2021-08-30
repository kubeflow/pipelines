/*
 * Copyright 2021 The Kubeflow Authors
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

import Button from '@material-ui/core/Button';
import FormControl from '@material-ui/core/FormControl';
import InputLabel from '@material-ui/core/InputLabel';
import MenuItem from '@material-ui/core/MenuItem';
import Paper from '@material-ui/core/Paper';
import Select from '@material-ui/core/Select';
import InfoIcon from '@material-ui/icons/InfoOutlined';
import * as React from 'react';
import { useState } from 'react';
import { ApiPipeline, ApiPipelineVersion } from 'src/apis/pipeline';
import { BannerProps } from 'src/components/Banner';
import { classes, stylesheet } from 'typestyle';
import MD2Tabs from '../atoms/MD2Tabs';
import { Description } from '../components/Description';
import Editor from '../components/Editor';
import Graph from '../components/Graph';
import ReduceGraphSwitch from '../components/ReduceGraphSwitch';
import SidePanel from '../components/SidePanel';
import StaticNodeDetails from '../components/StaticNodeDetails';
import { color, commonCss, fonts, fontsize, padding, zIndex } from '../Css';
import * as StaticGraphParser from '../lib/StaticGraphParser';
import { formatDateString, logger } from '../lib/Utils';

const summaryCardWidth = 500;

export const css = stylesheet({
  containerCss: {
    $nest: {
      '& .CodeMirror': {
        height: '100%',
        width: '80%',
      },

      '& .CodeMirror-gutters': {
        backgroundColor: '#f7f7f7',
      },
    },
    background: '#f7f7f7',
    height: '100%',
  },
  footer: {
    background: color.graphBg,
    display: 'flex',
    padding: '0 0 20px 20px',
  },
  footerInfoOffset: {
    marginLeft: summaryCardWidth + 40,
  },
  infoSpan: {
    color: color.lowContrast,
    fontFamily: fonts.secondary,
    fontSize: fontsize.small,
    letterSpacing: '0.21px',
    lineHeight: '24px',
    paddingLeft: 6,
  },
  summaryCard: {
    bottom: 20,
    left: 20,
    padding: 10,
    position: 'absolute',
    width: summaryCardWidth,
    zIndex: zIndex.PIPELINE_SUMMARY_CARD,
  },
  summaryKey: {
    color: color.strong,
    marginTop: 10,
  },
});

export interface PipelineDetailsV1Props {
  graph: dagre.graphlib.Graph | null;
  reducedGraph: dagre.graphlib.Graph | null;
  pipeline: ApiPipeline | null;
  selectedVersion: ApiPipelineVersion | undefined;
  versions: ApiPipelineVersion[];
  templateString?: string;
  updateBanner: (bannerProps: BannerProps) => void;
  handleVersionSelected: (versionId: string) => Promise<void>;
}

const PipelineDetailsV1: React.FC<PipelineDetailsV1Props> = ({
  pipeline,
  selectedVersion,
  versions,
  graph,
  reducedGraph,
  templateString,
  updateBanner,
  handleVersionSelected,
}: PipelineDetailsV1Props) => {
  const [selectedTab, setSelectedTab] = useState(0);
  const [selectedNodeId, setSelectedNodeId] = useState('');
  const [summaryShown, setSummaryShown] = useState(true);
  const [showReducedGraph, setShowReducedGraph] = useState(false);
  const graphToShow = showReducedGraph && reducedGraph ? reducedGraph : graph;

  let selectedNodeInfo: StaticGraphParser.SelectedNodeInfo | null = null;
  if (graphToShow && graphToShow.node(selectedNodeId)) {
    selectedNodeInfo = graphToShow.node(selectedNodeId).info;
    if (!!selectedNodeId && !selectedNodeInfo) {
      logger.error(`Node with ID: ${selectedNodeId} was not found in the graph`);
    }
  }

  // Since react-ace Editor doesn't support in Safari when height or width is a percentage.
  // Fix the Yaml file cannot display issue via defining “width/height” does not not take percentage if it's Safari browser.
  // The code of detecting wether isSafari is from: https://stackoverflow.com/questions/9847580/how-to-detect-safari-chrome-ie-firefox-and-opera-browser/9851769#9851769
  const isSafari =
    /constructor/i.test(window.HTMLElement.toString()) ||
    (function(p) {
      return p.toString() === '[object SafariRemoteNotification]';
    })(!window['safari'] || (typeof 'safari' !== 'undefined' && window['safari'].pushNotification));
  let editorHeightWidth = '100%';

  if (isSafari) {
    editorHeightWidth = '640px';
  }

  const createVersionUrl = () => {
    return selectedVersion!.code_source_url!;
  };

  return (
    <div className={commonCss.page} data-testid={'pipeline-detail-v1'}>
      <MD2Tabs
        selectedTab={selectedTab}
        onSwitch={(tab: number) => setSelectedTab(tab)}
        tabs={['Graph', 'YAML']}
      />
      <div className={commonCss.page}>
        {selectedTab === 0 && (
          <div className={commonCss.page}>
            {graphToShow && (
              <div className={commonCss.page} style={{ position: 'relative', overflow: 'hidden' }}>
                {!!pipeline && summaryShown && (
                  <Paper className={css.summaryCard}>
                    <div
                      style={{
                        alignItems: 'baseline',
                        display: 'flex',
                        justifyContent: 'space-between',
                      }}
                    >
                      <div className={commonCss.header}>Summary</div>
                      <Button onClick={() => setSummaryShown(false)} color='secondary'>
                        Hide
                      </Button>
                    </div>
                    <div className={css.summaryKey}>ID</div>
                    <div>{pipeline.id || 'Unable to obtain Pipeline ID'}</div>
                    {versions.length && (
                      <React.Fragment>
                        <form autoComplete='off'>
                          <FormControl>
                            <InputLabel>Version</InputLabel>
                            <Select
                              data-testid='version_selector'
                              value={
                                selectedVersion ? selectedVersion.id : pipeline.default_version!.id!
                              }
                              onChange={event => handleVersionSelected(event.target.value)}
                              inputProps={{ id: 'version-selector', name: 'selectedVersion' }}
                            >
                              {versions.map((v, _) => (
                                <MenuItem key={v.id} value={v.id}>
                                  {v.name}
                                </MenuItem>
                              ))}
                            </Select>
                          </FormControl>
                        </form>
                        <div className={css.summaryKey}>
                          <a href={createVersionUrl()} target='_blank' rel='noopener noreferrer'>
                            Version source
                          </a>
                        </div>
                      </React.Fragment>
                    )}
                    <div className={css.summaryKey}>Uploaded on</div>
                    <div>{formatDateString(pipeline.created_at)}</div>
                    <div className={css.summaryKey}>Description</div>
                    <Description description={pipeline.description || ''} />
                  </Paper>
                )}

                <Graph
                  graph={graphToShow}
                  selectedNodeId={selectedNodeId}
                  onClick={id => setSelectedNodeId(id)}
                  onError={(message, additionalInfo) => {
                    updateBanner({ message, additionalInfo, mode: 'error' });
                  }}
                />

                <ReduceGraphSwitch
                  disabled={!reducedGraph}
                  checked={showReducedGraph}
                  onChange={_ => {
                    setShowReducedGraph(!showReducedGraph);
                  }}
                />

                <SidePanel
                  isOpen={!!selectedNodeId}
                  title={selectedNodeId}
                  onClose={() => setSelectedNodeId('')}
                >
                  <div className={commonCss.page}>
                    {!selectedNodeInfo && (
                      <div className={commonCss.absoluteCenter}>Unable to retrieve node info</div>
                    )}
                    {!!selectedNodeInfo && (
                      <div className={padding(20, 'lr')}>
                        <StaticNodeDetails nodeInfo={selectedNodeInfo} />
                      </div>
                    )}
                  </div>
                </SidePanel>
                <div className={css.footer}>
                  {!summaryShown && (
                    <Button onClick={() => setSummaryShown(!summaryShown)} color='secondary'>
                      Show summary
                    </Button>
                  )}
                  <div
                    className={classes(
                      commonCss.flex,
                      summaryShown && !!pipeline && css.footerInfoOffset,
                    )}
                  >
                    <InfoIcon className={commonCss.infoIcon} />
                    <span className={css.infoSpan}>Static pipeline graph</span>
                  </div>
                </div>
              </div>
            )}
            {!graphToShow && <span style={{ margin: '40px auto' }}>No graph to show</span>}
          </div>
        )}
        {selectedTab === 1 && !!templateString && (
          <div className={css.containerCss} data-testid={'spec-yaml'}>
            <Editor
              value={templateString || ''}
              height={editorHeightWidth}
              width={editorHeightWidth}
              mode='yaml'
              theme='github'
              editorProps={{ $blockScrolling: true }}
              readOnly={true}
              highlightActiveLine={true}
              showGutter={true}
            />
          </div>
        )}
      </div>
    </div>
  );
};

export default PipelineDetailsV1;
