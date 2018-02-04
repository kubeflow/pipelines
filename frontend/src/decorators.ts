export function customElement(clazz: any) {
  const tagName = clazz.name.replace(/([a-z])([A-Z])/g, '$1-$2').toLowerCase();
  Object.defineProperty(clazz, 'is', {
    get: () => { return tagName; },
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
      type,
      notify: false,
      reflectToAttribute: false,
      readOnly: false,
    };
  };
}
