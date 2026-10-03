/*
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

import type { Meta, StoryObj } from '@storybook/react';
import { Plus } from 'lucide-react';
import { useId, useState } from 'react';
import { ThemeProvider } from '../modernization/ThemeProvider';
import { Button } from './button';
import { Input } from './input';

function Primitives() {
  const id = useId();
  const [name, setName] = useState('Training run');
  const [saved, setSaved] = useState('');
  return (
    <div className='min-h-screen p-8 font-kfp-sans'>
      <h1 className='mb-2 text-[22px] font-semibold tracking-[-0.02em]'>Component foundation</h1>
      <p className='mb-8 text-muted-foreground'>
        Controls, typography, and semantic colors for pipeline workflows.
      </p>
      <section
        className='mb-6 max-w-3xl rounded-panel border border-border bg-card p-6'
        aria-label='Buttons'
      >
        <h2 className='mb-4 text-[16px] font-semibold'>Actions</h2>
        <div className='flex flex-wrap gap-3'>
          <Button>
            <Plus aria-hidden='true' />
            New run
          </Button>
          <Button variant='secondary'>Compare</Button>
          <Button variant='ghost'>Cancel</Button>
          <Button variant='destructive'>Delete</Button>
          <Button disabled>Unavailable</Button>
          <Button disabled aria-busy='true'>
            Starting…
          </Button>
        </div>
      </section>
      <form
        className='max-w-3xl rounded-panel border border-border bg-card p-6'
        onSubmit={(event) => {
          event.preventDefault();
          setSaved(name);
        }}
      >
        <h2 className='mb-4 text-[16px] font-semibold'>Inputs</h2>
        <label className='mb-2 block font-medium' htmlFor={id}>
          Run name
        </label>
        <Input
          id={id}
          value={name}
          onChange={(event) => setName(event.target.value)}
          aria-describedby={`${id}-hint`}
          required
        />
        <p id={`${id}-hint`} className='mt-2 text-[12px] text-muted-foreground'>
          This story stores the name only in its preview state.
        </p>
        <div className='mt-4 flex items-center gap-4'>
          <Button type='submit'>Save preview</Button>
          <span role='status'>{saved ? `Saved “${saved}”` : ''}</span>
        </div>
        <label className='mb-2 mt-6 block font-medium' htmlFor={`${id}-invalid`}>
          Pipeline root
        </label>
        <Input
          id={`${id}-invalid`}
          defaultValue='unsupported://'
          aria-invalid='true'
          aria-describedby={`${id}-error`}
        />
        <p id={`${id}-error`} className='mt-2 text-[12px] text-destructive'>
          Enter a supported storage URI.
        </p>
        <p className='mt-6 font-kfp-mono text-[12px] text-foreground-2'>
          run-7f3a2c · 00:03:42 · learning_rate=0.01
        </p>
      </form>
    </div>
  );
}
const meta = {
  title: 'Modernization/Primitives',
  component: Primitives,
  parameters: { layout: 'fullscreen' },
  decorators: [
    (Story, context) => (
      <ThemeProvider
        defaultTheme={context.parameters.theme === 'dark' ? 'dark' : 'light'}
        storageKey={`kfp.storybook.primitives.${context.id}`}
      >
        <Story />
      </ThemeProvider>
    ),
  ],
} satisfies Meta<typeof Primitives>;
export default meta;
type Story = StoryObj<typeof meta>;
export const Light: Story = {};
export const Dark: Story = { parameters: { theme: 'dark' } };
