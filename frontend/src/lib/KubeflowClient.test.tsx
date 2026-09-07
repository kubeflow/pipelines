/**
 * Copyright 2026 The Kubeflow Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { render, screen } from '@testing-library/react';
import { NamespaceContext, useNamespaceChangeEvent } from './KubeflowClient';

function NamespaceChangeProbe() {
  const namespaceChanged = useNamespaceChangeEvent();
  return <span data-testid='namespace-changed'>{String(namespaceChanged)}</span>;
}

describe('useNamespaceChangeEvent', () => {
  it('reports a namespace transition for one render', () => {
    const view = render(
      <NamespaceContext.Provider value='ns1'>
        <NamespaceChangeProbe />
      </NamespaceContext.Provider>,
    );
    expect(screen.getByTestId('namespace-changed')).toHaveTextContent('false');

    view.rerender(
      <NamespaceContext.Provider value='ns2'>
        <NamespaceChangeProbe />
      </NamespaceContext.Provider>,
    );
    expect(screen.getByTestId('namespace-changed')).toHaveTextContent('true');

    view.rerender(
      <NamespaceContext.Provider value='ns2'>
        <NamespaceChangeProbe />
      </NamespaceContext.Provider>,
    );
    expect(screen.getByTestId('namespace-changed')).toHaveTextContent('false');
  });
});
