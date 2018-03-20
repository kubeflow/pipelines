/// <reference path="../../bower_components/polymer/types/polymer.d.ts" />

export interface PageElement extends HTMLElement {
  refresh(path: string, queryParams: {}): void;
}
