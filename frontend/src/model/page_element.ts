/// <reference path="../../bower_components/polymer/types/polymer.d.ts" />

export interface PageElement {
  load(path: string, queryParams: {}, data?: {}): void;
}
