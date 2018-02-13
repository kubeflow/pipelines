/// <reference path="../../bower_components/polymer/types/polymer.d.ts" />

export interface PageElement {
  refresh(path: string, queryParams: {}): void;
}
