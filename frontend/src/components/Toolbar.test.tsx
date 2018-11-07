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
import { createBrowserHistory } from 'history';
import Toolbar from './Toolbar';
import HelpIcon from '@material-ui/icons/Help';
import InfoIcon from '@material-ui/icons/Info';

const action1 = jest.fn();
const action2 = jest.fn();
const actions = [
  {
    action: action1,
    disabledTitle: 'test disabled title',
    icon: HelpIcon,
    id: 'test id',
    title: 'test title',
    tooltip: 'test tooltip',
  },
  {
    action: action2,
    disabled: true,
    disabledTitle: 'test disabled title2',
    icon: InfoIcon,
    id: 'test id2',
    title: 'test title2',
    tooltip: 'test tooltip2',
  },
];

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
  it('renders nothing when there are no breadcrumbs or actions', () => {
    const tree = shallow(<Toolbar breadcrumbs={[]} actions={[]} history={history} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders without breadcrumbs and one action', () => {
    const tree = shallow(<Toolbar breadcrumbs={[]} actions={[actions[0]]} history={history} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders without actions and one breadcrumb', () => {
    const tree = shallow(<Toolbar breadcrumbs={[breadcrumbs[0]]} actions={[]} history={history} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders without actions, one breadcrumb, and a page name', () => {
    const tree = shallow(<Toolbar breadcrumbs={breadcrumbs} actions={[]} history={history} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders without breadcrumbs and two actions', () => {
    const tree = shallow(<Toolbar breadcrumbs={[]} actions={actions} history={history} />);
    expect(tree).toMatchSnapshot();
  });

  it('fires the right action function when button is clicked', () => {
    const tree = shallow(<Toolbar breadcrumbs={[]} actions={actions} history={history} />);
    tree.find('BusyButton').at(0).simulate('click');
    expect(action1).toHaveBeenCalled();
    action2.mockClear();
  });

  it('renders with two breadcrumbs and two actions', () => {
    const tree = shallow(<Toolbar breadcrumbs={breadcrumbs} actions={actions} history={history} />);
    expect(tree).toMatchSnapshot();
  });
});
