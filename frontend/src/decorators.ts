export function customElement(clazz: any) {
  const tagName = clazz.name.replace(/([a-z])([A-Z])/g, '$1-$2').toLowerCase();
  Object.defineProperty(clazz, 'is', {
    get: () => { return tagName; },
  });

  window.customElements.define(tagName, clazz);
}
