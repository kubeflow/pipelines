import 'iron-icons/editor-icons';
import 'iron-icons/iron-icons';
import 'polymer/polymer-element.html';
import 'polymer/polymer.html';

import { customElement, observe, property } from '../../decorators';

import './file-browser.html';

import * as Apis from '../../lib/apis';
import * as Utils from '../../lib/utils';

export class FileClickEvent extends MouseEvent {
  public model: {
    file: FileDescriptor,
    index: number,
    itemsIndex: number,
  };
}

export class CrumbClickEvent extends MouseEvent {
  public model: {
    crumb: Breadcrumb,
    index: number,
    itemsIndex: number,
  };
}

export interface FileDescriptor {
  icon?: string;
  isDirectory: boolean;
  name: string;
  selected: boolean;
}

interface Breadcrumb {
  name: string;
  path: string;
}

@customElement
export class FileBrowser extends Polymer.Element {

  @property({ type: String })
  path = '';

  @property({ type: Array })
  breadcrumbs: Breadcrumb[] = [];

  @property({ type: Array })
  files: FileDescriptor[] = [];

  @observe('path')
  protected async _pathChanged(newPath: string) {
    if (newPath) {
      this.files = await Apis.listFiles(newPath);
      this.files.forEach((f) => {
        f.icon = f.isDirectory ? 'folder' : 'editor:insert-drive-file';
        f.selected = false;
      });
      (Polymer.dom as any).flush();
      (this.shadowRoot as ShadowRoot).querySelectorAll('.file').forEach(
        (e) => e.classList.remove('selected'));

      const parts = this.path.split('/').filter((p) => !!p);
      this.breadcrumbs = [];
      parts.forEach((p, i) => {
        this.breadcrumbs.push({
          name: p,
          path: '/' + parts.slice(0, i + 1).join('/'),
        });
      });
    }
  }

  protected _fileClicked(e: FileClickEvent) {
    const target = e.target as HTMLElement;
    const index = e.model.itemsIndex;
    this._unselectAllFiles();
    const oldSelected = this.files[index].selected;
    this.set(`files.${index}.selected`, !oldSelected);
    const element = Utils.getAncestorElementWithClass(target, 'file');
    element.classList.toggle('selected', !oldSelected);
  }

  protected _fileDoubleClicked(e: FileClickEvent) {
    const file = e.model.file;
    if (file.isDirectory) {
      const name = file.name;
      this.path += '/' + name;
    }
  }

  protected _crumbClicked(e: CrumbClickEvent) {
    this.path = e.model.crumb.path;
  }

  private _unselectAllFiles() {
    for (let i = 0; i < this.files.length; ++i) {
      this.set(`files.${i}.selected`, false);
    }
    (this.shadowRoot as ShadowRoot).querySelectorAll('.file').forEach(
      (f) => f.classList.remove('selected')
    );
  }
}
