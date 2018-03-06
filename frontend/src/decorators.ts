export function customElement(tagname: string) {
  return (clazz: any) => {
    Object.defineProperty(clazz, 'is', { value: tagname });
    window.customElements.define(clazz.is!, clazz);
  };
}

export interface PropertyOptions {
  type: any;
}

export function property(options: PropertyOptions) {
  return (proto: any, propName: any) => {
    const type = options.type;
    if (!proto.constructor.hasOwnProperty('properties')) {
      proto.constructor.properties = {};
    }
    proto.constructor.properties[propName] = {
      readOnly: false,
      reflectToAttribute: false,
      type,
    };
  };
}

export function observe(...targets: string[]) {
  return (proto: any, propName: string): any => {
    if (!proto.constructor.hasOwnProperty('observers')) {
      proto.constructor.observers = [];
    }
    proto.constructor.observers.push(`${propName}(${targets.join(',')})`);
  };
}
