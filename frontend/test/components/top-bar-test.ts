import 'paper-button/paper-button.html';
import './top-bar-test.html';

import { TopBar } from './top-bar';

declare function fixture<T>(element: string): T;
let testElement: TopBar;
let shadowRoot: ShadowRoot;

describe('top-bar', () => {
  beforeEach(() => {
    testElement = fixture<TopBar>('testFixture');
    shadowRoot = testElement.shadowRoot as ShadowRoot;
  });

  it('should include logo', () => {
    const logo = testElement.querySelector('#logo') as HTMLAnchorElement;
    assert.equal(logo.tagName, 'A');
    assert.equal(logo.innerHTML, 'ML Pipeline Management Console');
  });

  it('redirects to packages page when logo is pressed', () => {
    const logo = testElement.querySelector('#logo') as HTMLAnchorElement;
    assert.equal(logo.href, '/packages');
  });

  it('should include package list button', () => {
    const packagesBtn =
      shadowRoot.querySelector('#packagesBtn') as HTMLAnchorElement;
    assert.equal(packagesBtn.innerText, 'PIPELINES GALLERY');
    assert.equal(packagesBtn.href, '/packages');
  });
});
