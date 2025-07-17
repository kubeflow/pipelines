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
import { render, screen, fireEvent } from '@testing-library/react';
import '@testing-library/jest-dom';
import { createBrowserHistory, createMemoryHistory } from 'history';
import Toolbar, { ToolbarActionMap } from './Toolbar';
import HelpIcon from '@mui/icons-material/Help';
import InfoIcon from '@mui/icons-material/Info';

const action1 = jest.fn();
const action2 = jest.fn();
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

const history = createBrowserHistory({});

describe('Toolbar', () => {
  beforeAll(() => {
    history.push('/pipelines');
  });

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('renders nothing when there are no breadcrumbs or actions', () => {
    const { container } = render(<Toolbar breadcrumbs={[]} actions={{}} history={history} pageTitle='' />);
    expect(container.firstChild).toBeEmptyDOMElement();
  });

  it('renders without breadcrumbs and a string page title', () => {
    render(
      <Toolbar breadcrumbs={[]} actions={actions} history={history} pageTitle='test page title' />,
    );
    expect(screen.getByText('test page title')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /test title/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /test title2/i })).toBeInTheDocument();
  });

  it('renders without breadcrumbs and a component page title', () => {
    render(
      <Toolbar
        breadcrumbs={[]}
        actions={actions}
        history={history}
        pageTitle={<div id='myComponent'>test page title</div>}
      />,
    );
    expect(screen.getByText('test page title')).toBeInTheDocument();
    expect(document.getElementById('myComponent')).toBeInTheDocument();
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
    render(
      <Toolbar
        breadcrumbs={[]}
        actions={singleAction}
        history={history}
        pageTitle='test page title'
      />,
    );
    expect(screen.getByRole('button', { name: /test title/i })).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /test title2/i })).not.toBeInTheDocument();
  });

  it('renders without actions and one breadcrumb', () => {
    render(
      <Toolbar
        breadcrumbs={[breadcrumbs[0]]}
        actions={{}}
        history={history}
        pageTitle='test page title'
      />,
    );
    expect(screen.getByText('test display name')).toBeInTheDocument();
    expect(screen.getByText('test page title')).toBeInTheDocument();
  });

  it('renders without actions, one breadcrumb, and a page name', () => {
    render(
      <Toolbar
        breadcrumbs={[breadcrumbs[0]]}
        actions={{}}
        history={history}
        pageTitle='test page title'
      />,
    );
    expect(screen.getByText('test display name')).toBeInTheDocument();
    expect(screen.getByText('test page title')).toBeInTheDocument();
  });

  it('renders without breadcrumbs and two actions', () => {
    render(
      <Toolbar breadcrumbs={[]} actions={actions} history={history} pageTitle='test page title' />,
    );
    expect(screen.getByRole('button', { name: /test title/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /test title2/i })).toBeInTheDocument();
  });

  it('fires the right action function when button is clicked', () => {
    render(
      <Toolbar breadcrumbs={[]} actions={actions} history={history} pageTitle='test page title' />,
    );
    const button = screen.getByRole('button', { name: /test title/i });
    fireEvent.click(button);
    expect(action1).toHaveBeenCalled();
  });

  it('renders outlined action buttons', () => {
    const outlinedActions = {
      action1: {
        action: jest.fn(),
        id: 'test outlined id',
        outlined: true,
        title: 'test outlined title',
        tooltip: 'test outlined tooltip',
      },
    };

    render(
      <Toolbar
        breadcrumbs={breadcrumbs}
        actions={outlinedActions}
        pageTitle=''
        history={history}
      />,
    );
    expect(screen.getByRole('button', { name: /test outlined title/i })).toBeInTheDocument();
  });

  it('renders primary action buttons', () => {
    const primaryActions = {
      action1: {
        action: jest.fn(),
        id: 'test primary id',
        primary: true,
        title: 'test primary title',
        tooltip: 'test primary tooltip',
      },
    };

    render(
      <Toolbar breadcrumbs={breadcrumbs} actions={primaryActions} pageTitle='' history={history} />,
    );
    expect(screen.getByRole('button', { name: /test primary title/i })).toBeInTheDocument();
  });

  it('renders primary action buttons without outline, even if outline is true', () => {
    const outlinedPrimaryActions = {
      action1: {
        action: jest.fn(),
        id: 'test id',
        outlined: true,
        primary: true,
        title: 'test title',
        tooltip: 'test tooltip',
      },
    };

    render(
      <Toolbar
        breadcrumbs={breadcrumbs}
        actions={outlinedPrimaryActions}
        pageTitle=''
        history={history}
      />,
    );
    expect(screen.getByRole('button', { name: /test title/i })).toBeInTheDocument();
  });

  it('renders with two breadcrumbs and two actions', () => {
    render(
      <Toolbar breadcrumbs={breadcrumbs} actions={actions} pageTitle='' history={history} />,
    );
    expect(screen.getByText('test display name')).toBeInTheDocument();
    expect(screen.getByText('test display name2')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /test title/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /test title2/i })).toBeInTheDocument();
  });

  it('disables the back button when there is no browser history', () => {
    // This test uses createMemoryHistory because createBroweserHistory returns a singleton, and
    // there is no way to clear its entries which this test requires.
    const emptyHistory = createMemoryHistory();
    render(
      <Toolbar breadcrumbs={breadcrumbs} actions={actions} history={emptyHistory} pageTitle='' />,
    );
    const backButton = screen.getByLabelText(/go back/i);
    expect(backButton).toBeDisabled();
  });
});
