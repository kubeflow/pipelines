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
import Banner, { css } from './Banner';

jest.mock('react-i18next', () => ({
  withTranslation: () => (Component: { defaultProps: any }) => {
    Component.defaultProps = { ...Component.defaultProps, t: ((key: string) => key) as any };
    return Component;
  },
}));
describe('Banner', () => {
  it('defaults to error mode', () => {
    const tree = shallow(<Banner message={'Some message'} />);
    expect(tree).toMatchSnapshot();
  });

  it('uses error mode when instructed', () => {
    const tree = shallow(<Banner message={'Some message'} mode={'error'} />);
    expect(tree).toMatchSnapshot();
  });

  it('uses warning mode when instructed', () => {
    const tree = shallow(<Banner message={'Some message'} mode={'warning'} />);
    expect(tree).toMatchSnapshot();
  });

  it('uses info mode when instructed', () => {
    const tree = shallow(<Banner message={'Some message'} mode={'info'} />);
    expect(tree).toMatchSnapshot();
  });

  it('shows "Details" button and has dialog when there is additional info', () => {
    const tree = shallow(<Banner message={'Some message'} additionalInfo={'More info'} />);
    expect(tree).toMatchSnapshot();
  });

  it('shows "Refresh" button when passed a refresh function', () => {
    const tree = shallow(
      <Banner
        message={'Some message'}
        refresh={() => {
          /* do nothing */
        }}
      />,
    );
    expect(tree).toMatchSnapshot();
  });

  it('does not show "Refresh" button if mode is "info"', () => {
    const tree = shallow(
      <Banner
        message={'Some message'}
        mode={'info'}
        refresh={() => {
          /* do nothing */
        }}
      />,
    );
    expect(tree.findWhere(el => el.text() === 'Refresh').exists()).toEqual(false);
  });

  it('shows troubleshooting link instructed by prop', () => {
    const tree = shallow(
      <Banner message='Some message' mode='error' showTroubleshootingGuideLink={true} />,
    );
    expect(tree).toMatchInlineSnapshot(`
      <div
        className="flex banner mode"
      >
        <div
          className="message"
        >
          <pure(ErrorIcon)
            className="icon"
          />
          Some message
        </div>
        <div
          className="flex"
        >
          <a
            className="troubleShootingLink"
            href="https://www.kubeflow.org/docs/pipelines/troubleshooting"
          >
            common:troubleshooting
          </a>
        </div>
      </div>
    `);
  });

  it('does not show troubleshooting link if warning', () => {
    const tree = shallow(
      <Banner message='Some message' mode='warning' showTroubleshootingGuideLink={true} />,
    );
    expect(tree.findWhere(el => el.text() === 'Troubleshooting guide').exists()).toEqual(false);
  });

  it('opens details dialog when button is clicked', () => {
    const tree = shallow(<Banner message='hello' additionalInfo='world' />);
    tree
      .find('WithStyles(Button)')
      .at(0)
      .simulate('click');
    expect(tree.state()).toHaveProperty('dialogOpen', true);
  });

  it('closes details dialog when cancel button is clicked', () => {
    const tree = shallow(<Banner message='hello' additionalInfo='world' />);
    tree
      .find('WithStyles(Button)')
      .at(0)
      .simulate('click');
    expect(tree.state()).toHaveProperty('dialogOpen', true);
    tree.find('#dismissDialogBtn').simulate('click');
    expect(tree.state()).toHaveProperty('dialogOpen', false);
  });

  it('calls refresh callback', () => {
    const spy = jest.fn();
    const tree = shallow(<Banner message='hello' refresh={spy} />);
    tree
      .find('.' + css.refreshButton)
      .at(0)
      .simulate('click');
    expect(spy).toHaveBeenCalled();
  });
});
