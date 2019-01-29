/*
 * Copyright 2018 Google LLC
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

import AddIcon from '@material-ui/icons/Add';
import CollapseIcon from '@material-ui/icons/UnfoldLess';
import ExpandIcon from '@material-ui/icons/UnfoldMore';
import { ToolbarActionConfig } from '../components/Toolbar';

interface ButtonsMap { [key: string]: (action: () => void) => ToolbarActionConfig; }

// tslint:disable-next-line:variable-name
const Buttons: ButtonsMap = {
  archive: action => ({
    action,
    disabled: true,
    disabledTitle: 'Select at least one resource to archive',
    id: 'archiveBtn',
    title: 'Archive',
    tooltip: 'Archive',
  }),
  cloneRun: action => ({
    action,
    disabled: true,
    disabledTitle: 'Select a run to clone',
    id: 'cloneBtn',
    title: 'Clone run',
    tooltip: 'Create a copy from this run\s initial state',
  }),
  collapseSections: action => ({
    action,
    icon: CollapseIcon,
    id: 'collapseBtn',
    title: 'Collapse all',
    tooltip: 'Collapse all sections',
  }),
  compareRuns: action => ({
    action,
    disabled: true,
    disabledTitle: 'Select multiple runs to compare',
    id: 'compareBtn',
    title: 'Compare runs',
    tooltip: 'Compare up to 10 selected runs',
  }),
  delete: action => ({
    action,
    disabled: true,
    disabledTitle: 'Select at least one resource to delete',
    id: 'deleteBtn',
    title: 'Delete',
    tooltip: 'Delete',
  }),
  disableRun: action => ({
    action,
    disabled: true,
    disabledTitle: 'Run schedule already disabled',
    id: 'disableBtn',
    title: 'Disable',
    tooltip: 'Disable the run\'s trigger',
  }),
  enableRun: action => ({
    action,
    disabled: true,
    disabledTitle: 'Run schedule already enabled',
    id: 'enableBtn',
    title: 'Enable',
    tooltip: 'Enable the run\'s trigger',
  }),
  expandSections: action => ({
    action,
    icon: ExpandIcon,
    id: 'expandBtn',
    title: 'Expand all',
    tooltip: 'Expand all sections',
  }),
  newExperiment: action => ({
    action,
    icon: AddIcon,
    id: 'newExperimentBtn',
    outlined: true,
    title: 'Create an experiment',
    tooltip: 'Create a new experiment',
  }),
  newRun: action => ({
    action,
    icon: AddIcon,
    id: 'createNewRunBtn',
    outlined: true,
    primary: true,
    title: 'Create run',
    tooltip: 'Create a new run within this pipeline',
  }),
  refresh: action => ({
    action,
    id: 'refreshBtn',
    title: 'Refresh',
    tooltip: 'Refresh the list',
  }),
  restore: action => ({
    action,
    disabled: true,
    disabledTitle: 'Select at least one resource to restore',
    id: 'archiveBtn',
    title: 'Archive',
    tooltip: 'Archive',
  }),
  upload: action => ({
    action,
    icon: AddIcon,
    id: 'uploadBtn',
    outlined: true,
    title: 'Upload pipeline',
    tooltip: 'Upload pipeline',
  }),
};

export default Buttons;
