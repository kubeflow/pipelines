import { MyTopBar } from './top-bar';

// add custom elements here
const elements = {
  MyTopBar,
};

function toKebabCase(name: string) {
  return name.replace(/([a-z])([A-Z])/g, '$1-$2').toLowerCase();
}

// register all components as kebab case
Object.keys(elements)
  .forEach(key => {
    customElements.define(toKebabCase(key), elements[key])
  });
