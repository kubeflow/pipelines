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

import * as React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { createMemoryHistory } from 'history';
import { vi } from 'vitest';
import Toolbar, { ToolbarActionMap } from './Toolbar';
import HelpIcon from '@material-ui/icons/Help';
import InfoIcon from '@material-ui/icons/Info';

const action1 = vi.fn();
const action2 = vi.fn();
const actions: ToolbarActionMap = {
  action1: {
    action: action1,
    disabledTitle: 'test disabled title',
    icon: HelpIcon,
    id: 'test id',
    title: 'test title',
    tooltip: 'test tooltip',
  },
  action2: {
    action: action2,
    disabled: true,
    disabledTitle: 'test disabled title2',
    icon: InfoIcon,
    id: 'test id2',
    title: 'test title2',
    tooltip: 'test tooltip2',
  },
};

const breadcrumbs = [
  {
    displayName: 'test display name',
    href: '/some/test/path',
  },
  {
    displayName: 'test display name2',
    href: '/some/test/path2',
  },
];

const history = createMemoryHistory();

function renderWithRouter(ui: React.ReactElement) {
  return render(<MemoryRouter>{ui}</MemoryRouter>);
}

describe('Toolbar', () => {
  beforeEach(() => {
    history.push('/pipelines');
    vi.clearAllMocks();
  });

  it('renders nothing when there are no breadcrumbs or actions', () => {
    const { container } = render(
      <Toolbar breadcrumbs={[]} actions={{}} history={history} pageTitle='' />,
    );
    expect(container.firstChild).toBeNull();
  });

  it('renders without breadcrumbs and a string page title', () => {
    const { asFragment } = renderWithRouter(
      <Toolbar breadcrumbs={[]} actions={actions} history={history} pageTitle='test page title' />,
    );
    expect(asFragment()).toMatchSnapshot();
  });

  it('renders without breadcrumbs and a component page title', () => {
    const { asFragment } = renderWithRouter(
      <Toolbar
        breadcrumbs={[]}
        actions={actions}
        history={history}
        pageTitle={<div id='myComponent'>test page title</div>}
      />,
    );
    expect(asFragment()).toMatchSnapshot();
  });

  it('renders without breadcrumbs and one action', () => {
    const singleAction = {
      action1: {
        action: action1,
        disabledTitle: 'test disabled title',
        icon: HelpIcon,
        id: 'test id',
        title: 'test title',
        tooltip: 'test tooltip',
      },
    };
    const { asFragment } = renderWithRouter(
      <Toolbar
        breadcrumbs={[]}
        actions={singleAction}
        history={history}
        pageTitle='test page title'
      />,
    );
    expect(asFragment()).toMatchSnapshot();
  });

  it('renders without actions and one breadcrumb', () => {
    const { asFragment } = renderWithRouter(
      <Toolbar
        breadcrumbs={[breadcrumbs[0]]}
        actions={{}}
        history={history}
        pageTitle='test page title'
      />,
    );
    expect(asFragment()).toMatchSnapshot();
  });

  it('renders without actions, one breadcrumb, and a page name', () => {
    const { asFragment } = renderWithRouter(
      <Toolbar
        breadcrumbs={[breadcrumbs[0]]}
        actions={{}}
        history={history}
        pageTitle='test page title'
      />,
    );
    expect(asFragment()).toMatchSnapshot();
  });

  it('renders without breadcrumbs and two actions', () => {
    const { asFragment } = renderWithRouter(
      <Toolbar breadcrumbs={[]} actions={actions} history={history} pageTitle='test page title' />,
    );
    expect(asFragment()).toMatchSnapshot();
  });

  it('fires the right action function when button is clicked', () => {
    renderWithRouter(
      <Toolbar breadcrumbs={[]} actions={actions} history={history} pageTitle='test page title' />,
    );
    fireEvent.click(screen.getByRole('button', { name: 'test title' }));
    expect(action1).toHaveBeenCalled();
  });

  it('renders outlined action buttons', () => {
    const outlinedActions = {
      action1: {
        action: vi.fn(),
        id: 'test outlined id',
        outlined: true,
        title: 'test outlined title',
        tooltip: 'test outlined tooltip',
      },
    };

    const { asFragment } = renderWithRouter(
      <Toolbar
        breadcrumbs={breadcrumbs}
        actions={outlinedActions}
        pageTitle=''
        history={history}
      />,
    );
    expect(asFragment()).toMatchSnapshot();
  });

  it('renders primary action buttons', () => {
    const primaryActions = {
      action1: {
        action: vi.fn(),
        id: 'test primary id',
        primary: true,
        title: 'test primary title',
        tooltip: 'test primary tooltip',
      },
    };

    const { asFragment } = renderWithRouter(
      <Toolbar breadcrumbs={breadcrumbs} actions={primaryActions} pageTitle='' history={history} />,
    );
    expect(asFragment()).toMatchSnapshot();
  });

  it('renders primary action buttons without outline, even if outline is true', () => {
    const outlinedPrimaryActions = {
      action1: {
        action: vi.fn(),
        id: 'test id',
        outlined: true,
        primary: true,
        title: 'test title',
        tooltip: 'test tooltip',
      },
    };

    const { asFragment } = renderWithRouter(
      <Toolbar
        breadcrumbs={breadcrumbs}
        actions={outlinedPrimaryActions}
        pageTitle=''
        history={history}
      />,
    );
    expect(asFragment()).toMatchSnapshot();
  });

  it('renders with two breadcrumbs and two actions', () => {
    const { asFragment } = renderWithRouter(
      <Toolbar breadcrumbs={breadcrumbs} actions={actions} pageTitle='' history={history} />,
    );
    expect(asFragment()).toMatchSnapshot();
  });
});
