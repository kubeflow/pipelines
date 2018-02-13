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

  it('redirects to templates page when logo is pressed', () => {
    const logo = testElement.querySelector('#logo') as HTMLAnchorElement;
    assert.equal(logo.href, '/templates');
  });

  it('should include template list button', () => {
    const templatesBtn =
      shadowRoot.querySelector('#templatesBtn') as HTMLAnchorElement;
    assert.equal(templatesBtn.innerText, 'PIPELINE TEMPLATES');
    assert.equal(templatesBtn.href, '/templates');
  });
});
