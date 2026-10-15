/*
 * Copyright 2018 The Kubeflow Authors
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

import { createRef } from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import { AutoLink, ExternalLink } from './ExternalLink';

it('preserves external href, callback, custom classes and ref while opening safely', () => {
  const click = vi.fn((event) => event.preventDefault());
  const ref = createRef<HTMLAnchorElement>();
  render(
    <ExternalLink href='https://example.test/docs' className='custom' onClick={click} ref={ref}>
      Documentation
    </ExternalLink>,
  );
  const link = screen.getByRole('link', { name: 'Documentation' });
  expect(link).toHaveAttribute('href', 'https://example.test/docs');
  expect(link).toHaveAttribute('target', '_blank');
  expect(link).toHaveAttribute('rel', 'noopener');
  expect(link).toHaveClass('kfp-external-link', 'custom');
  expect(ref.current).toBe(link);
  fireEvent.click(link);
  expect(click).toHaveBeenCalledTimes(1);
});

it('keeps hash navigation in the current tab and external links in a new tab', () => {
  render(
    <>
      <AutoLink href='#/pipelines/details/id'>Pipeline</AutoLink>
      <AutoLink href='https://example.test'>Website</AutoLink>
    </>,
  );
  expect(screen.getByRole('link', { name: 'Pipeline' })).not.toHaveAttribute('target');
  expect(screen.getByRole('link', { name: 'Pipeline' })).toHaveAttribute(
    'href',
    '#/pipelines/details/id',
  );
  expect(screen.getByRole('link', { name: 'Website' })).toHaveAttribute('target', '_blank');
});
