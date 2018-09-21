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

import 'codemirror/addon/display/autorefresh.js';
import 'codemirror/mode/yaml/yaml.js';
import 'iron-icons/iron-icons.html';
import 'paper-button/paper-button.html';
import 'paper-progress/paper-progress.html';
import 'paper-tabs/paper-tab.html';
import 'paper-tabs/paper-tabs.html';
import 'polymer/polymer.html';

import { EditorFromTextArea, fromTextArea } from 'codemirror';
import * as Apis from '../../lib/apis';
import * as Utils from '../../lib/utils';

import { customElement, observe, property } from 'polymer-decorators/src/decorators';
import { apiGetTemplateResponse, apiPipeline} from '../../api/pipeline';
import { DialogResult } from '../../components/popup-dialog/popup-dialog';
import { RouteEvent } from '../../model/events';
import { PageElement } from '../../model/page_element';

import './pipeline-details.html';

@customElement('pipeline-details')
export class PipelineDetails extends PageElement {

  @property({ type: Object })
  public pipeline?: apiPipeline;

  @property({ type: Number })
  public selectedTab = -1;

  @property({ type: Boolean })
  protected _busy = false;

  @property({ type: Boolean })
  protected _createJobDisable = true;

  private codeMirror?: EditorFromTextArea;

  private pipelineTemplate?: apiGetTemplateResponse;

  public get tabs(): PaperTabsElement {
    return this.$.tabs as PaperTabsElement;
  }

  public get createJobButton(): PaperButtonElement {
    return this.$.createJobBtn as PaperButtonElement;
  }

  public get deleteButton(): PaperButtonElement {
    return this.$.deleteBtn as PaperButtonElement;
  }

  public async load(id: string): Promise<void> {
    if (!!id) {
      const tabElement = this.tabs.querySelector(`[href="${location.hash}"]`);
      // TODO: change this from 1 to 0 once the graph page is ready.
      this.selectedTab = tabElement ? this.tabs.indexOf(tabElement) : 1;

      await this._loadPipeline(id);
    }
  }

  protected async _loadPipeline(id: string): Promise<void> {
    this._busy = true;
    try {
      await Promise.all([ Apis.getPipeline(id), Apis.getPipelineTemplate(id) ])
        .then(([ pipeline, template ]) => {
          this.pipeline = pipeline;
          this.pipelineTemplate = template;
        });
      this._createJobDisable = false;
    } catch (err) {
      this.showPageError(
          'There was an error while loading details for pipeline ' + id, err.message);
      Utils.log.verbose('Error loading pipeline:', err);
    } finally {
      this._busy = false;
    }

    this.codeMirror = fromTextArea(this.$.sourceCode as HTMLTextAreaElement, {
      lineNumbers: true,
      lineWrapping: true,
      mode: 'text/yaml',
      readOnly: 'nocursor',
      theme: 'default',
    });

    if (this.pipelineTemplate && this.pipelineTemplate.template) {
      this.codeMirror.setValue(this.pipelineTemplate.template);
    }

    // This addon ensures that the CodeMirror is refreshed when its containing element becomes
    // visible. Without it, the mirror does not display when navigating to it from another page,
    // unless the entire page is refreshed.
    this.codeMirror.setOption('autoRefresh', true);

    // Calling refresh here ensures that the CodeMirror loads properly when a user refreshes while
    // on the Pipeline details page. Unfortunately, it seems both this and the autoRefresh above are
    // needed.
    this.codeMirror.refresh();
  }

  protected _createJob(): void {
    if (this.pipeline) {
      this.dispatchEvent(
          new RouteEvent(
            '/jobs/new',
            {
              pipelineId: this.pipeline.id,
            }));
    }
  }

  protected async _deletePipeline(): Promise<void> {
    if (this.pipeline) {
      const dialogResult = await Utils.showDialog(
          'Delete pipeline?',
          'You are about to delete this pipeline. Are you sure you want to proceed?',
          'Delete pipeline',
          'Cancel');

      // BUTTON1 is Delete
      if (dialogResult !== DialogResult.BUTTON1) {
        return;
      }

      this._busy = true;
      try {
        await Apis.deletePipeline(this.pipeline.id!);

        Utils.showNotification(`Successfully deleted Pipeline: "${this.pipeline.name}"`);

        // Navigate back to Pipeline list page upon successful deletion.
        this.dispatchEvent(new RouteEvent('/pipelines'));
      } catch (err) {
        Utils.showDialog('Failed to delete Pipeline', err);
      } finally {
        this._busy = false;
      }
    }
  }

  @observe('selectedTab')
  protected _selectedTabChanged(newTab: number): void {
    if (this.selectedTab === -1) {
      return;
    }

    const tab = (this.tabs.selectedItem || this.tabs.children[0]) as PaperTabElement | undefined;
    if (tab) {
      location.hash = tab.getAttribute('href') || '';
    }
  }
}
