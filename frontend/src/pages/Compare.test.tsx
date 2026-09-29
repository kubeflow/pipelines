/*
 * Copyright 2022 The Kubeflow Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { render, screen } from '@testing-library/react';
import { QUERY_PARAMS } from 'src/components/Router';
import { PageProps } from './Page';
import Compare from './Compare';

vi.mock('./CompareV2', () => ({ default: () => <div>Native comparison</div> }));
function props(runIds: string[]): PageProps {
  return {
    navigate: vi.fn(),
    params: {},
    location: { search: `?${QUERY_PARAMS.runlist}=${runIds.join(',')}` } as any,
    toolbarProps: { actions: {}, breadcrumbs: [], pageTitle: '' },
    updateBanner: vi.fn(),
    updateDialog: vi.fn(),
    updateSnackbar: vi.fn(),
    updateToolbar: vi.fn(),
  };
}
it('renders the native comparison without a second routing fetch', () => {
  render(<Compare {...props(['run-1', 'run-2'])} />);
  expect(screen.getByText('Native comparison')).toBeInTheDocument();
});
it.each([0, 1, 11])('rejects %s selected runs before fetching', (count) => {
  const pageProps = props(Array.from({ length: count }, (_, i) => `${i}`));
  render(<Compare {...pageProps} />);
  expect(screen.queryByText('Native comparison')).toBeNull();
  expect(pageProps.updateBanner).toHaveBeenCalledWith(expect.objectContaining({ mode: 'error' }));
});
it('clears the count error when the selection becomes valid', () => {
  const pageProps = props(['run-1']);
  const { rerender } = render(<Compare {...pageProps} />);
  rerender(<Compare {...pageProps} location={props(['run-1', 'run-2']).location} />);
  expect(pageProps.updateBanner).toHaveBeenLastCalledWith({});
  expect(screen.getByText('Native comparison')).toBeInTheDocument();
});
