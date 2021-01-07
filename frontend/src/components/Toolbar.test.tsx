/*
 * Copyright 2018 Google LLC
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
import { shallow } from 'enzyme';
import { createBrowserHistory, createMemoryHistory } from 'history';
import Toolbar, { ToolbarActionMap } from './Toolbar';
import HelpIcon from '@material-ui/icons/Help';
import InfoIcon from '@material-ui/icons/Info';

jest.mock('react-i18next', () => ({
  // this mock makes sure any components using the translate hook can use it without a warning being shown
  withTranslation: () => (Component: { defaultProps: any }) => {
    Component.defaultProps = { ...Component.defaultProps, t: (key: string) => key };
    return Component;
  },
  useTranslation: () => {
    return {
      t: (key: string) => key,
    };
  },
}));
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

  it('renders nothing when there are no breadcrumbs or actions', () => {
    const tree = shallow(<Toolbar breadcrumbs={[]} actions={{}} history={history} pageTitle='' />);
    expect(tree).toMatchSnapshot();
  });

  it('renders without breadcrumbs and a string page title', () => {
    const tree = shallow(
      <Toolbar breadcrumbs={[]} actions={actions} history={history} pageTitle='test page title' />,
    );
    expect(tree).toMatchSnapshot();
  });

  it('renders without breadcrumbs and a component page title', () => {
    const tree = shallow(
      <Toolbar
        breadcrumbs={[]}
        actions={actions}
        history={history}
        pageTitle={<div id='myComponent'>test page title</div>}
      />,
    );
    expect(tree).toMatchSnapshot();
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
    const tree = shallow(
      <Toolbar
        breadcrumbs={[]}
        actions={singleAction}
        history={history}
        pageTitle='test page title'
      />,
    );
    expect(tree).toMatchSnapshot();
  });

  it('renders without actions and one breadcrumb', () => {
    const tree = shallow(
      <Toolbar
        breadcrumbs={[breadcrumbs[0]]}
        actions={{}}
        history={history}
        pageTitle='test page title'
      />,
    );
    expect(tree).toMatchSnapshot();
  });

  it('renders without actions, one breadcrumb, and a page name', () => {
    const tree = shallow(
      <Toolbar
        breadcrumbs={[breadcrumbs[0]]}
        actions={{}}
        history={history}
        pageTitle='test page title'
      />,
    );
    expect(tree).toMatchSnapshot();
  });

  it('renders without breadcrumbs and two actions', () => {
    const tree = shallow(
      <Toolbar breadcrumbs={[]} actions={actions} history={history} pageTitle='test page title' />,
    );
    expect(tree).toMatchSnapshot();
  });

  it('fires the right action function when button is clicked', () => {
    const tree = shallow(
      <Toolbar breadcrumbs={[]} actions={actions} history={history} pageTitle='test page title' />,
    );
    tree
      .find('BusyButton')
      .at(0)
      .simulate('click');
    expect(action1).toHaveBeenCalled();
    action1.mockClear();
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

    const tree = shallow(
      <Toolbar
        breadcrumbs={breadcrumbs}
        actions={outlinedActions}
        pageTitle=''
        history={history}
      />,
    );
    expect(tree).toMatchSnapshot();
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

    const tree = shallow(
      <Toolbar breadcrumbs={breadcrumbs} actions={primaryActions} pageTitle='' history={history} />,
    );
    expect(tree).toMatchSnapshot();
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

    const tree = shallow(
      <Toolbar
        breadcrumbs={breadcrumbs}
        actions={outlinedPrimaryActions}
        pageTitle=''
        history={history}
      />,
    );
    expect(tree).toMatchSnapshot();
  });

  it('renders with two breadcrumbs and two actions', () => {
    const tree = shallow(
      <Toolbar breadcrumbs={breadcrumbs} actions={actions} pageTitle='' history={history} />,
    );
    expect(tree).toMatchSnapshot();
  });

  it('disables the back button when there is no browser history', () => {
    // This test uses createMemoryHistory because createBroweserHistory returns a singleton, and
    // there is no way to clear its entries which this test requires.
    const emptyHistory = createMemoryHistory();
    const tree = shallow(
      <Toolbar breadcrumbs={breadcrumbs} actions={actions} history={emptyHistory} pageTitle='' />,
    );
    expect(tree).toMatchSnapshot();
  });
});
