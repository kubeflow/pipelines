export function customElement(clazz: any) {
  const tagName = clazz.name.replace(/([a-z])([A-Z])/g, '$1-$2').toLowerCase();
  Object.defineProperty(clazz, 'is', {
    get: () => tagName,
  });

  window.customElements.define(tagName, clazz);
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
      notify: false,
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
