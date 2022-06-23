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

import { render, screen, fireEvent } from '@testing-library/react';
import * as React from 'react';
import { testBestPractices } from 'src/TestUtils';
import TwoLevelDropdown, {
  DropdownItem,
  SelectedItem,
  TwoLevelDropdownProps,
} from './TwoLevelDropdown';

testBestPractices();
describe('TwoLevelDropdown', () => {
  const title = 'Dropdown title';
  const items: DropdownItem[] = [
    {
      name: 'first',
      subItems: [
        {
          name: 'second',
        },
        {
          name: 'third',
          secondaryName: 'fourth',
        },
      ],
    },
    {
      name: 'fifth',
      subItems: [
        {
          name: 'sixth',
          secondaryName: 'seventh',
        },
      ],
    },
  ];
  const selectedItem: SelectedItem = { itemName: '', subItemName: '' };
  const setSelectedItem: (selectedItem: SelectedItem) => void = jest.fn();

  function generateProps(): TwoLevelDropdownProps {
    return {
      title,
      items,
      selectedItem,
      setSelectedItem,
    } as TwoLevelDropdownProps;
  }

  it('Render two-level dropdown', async () => {
    render(<TwoLevelDropdown {...generateProps()} />);
    screen.getByText(title);
  });

  it('Render two-level dropdown with selection', async () => {
    const props: TwoLevelDropdownProps = generateProps();
    props.selectedItem = {
      itemName: 'first',
      subItemName: 'second',
      subItemSecondaryName: 'third'
    } as SelectedItem;
    render(<TwoLevelDropdown {...props} />);
    expect(screen.queryByText(title)).toBeNull();
    screen.getByText(/first/);
    screen.getByText(/second/);
    screen.getByText(/third/);
  });

  it('Two-level dropdown toggle behavior', async () => {
    render(<TwoLevelDropdown {...generateProps()} />);

    expect(screen.queryByText('first')).toBeNull();

    // Display the dropdown
    fireEvent.click(screen.getByText(title));
    screen.getByText('first');

    // Hide the dropdown
    fireEvent.click(screen.getByText(title));
    expect(screen.queryByText('first')).toBeNull();

    // Display the dropdown again
    fireEvent.click(screen.getByText(title));
    screen.getByText('first');

    // Click on dropdown text (not button) does not close dropdown
    fireEvent.click(screen.getByText('first'));
    screen.getByText('first');

    // Click outside closes the dropdown
    fireEvent.click(document.body);
    expect(screen.queryByText('first')).toBeNull();
  });

  it('Two-level dropdown sub items display', async () => {
    render(<TwoLevelDropdown {...generateProps()} />);

    fireEvent.click(screen.getByText(title));

    // First dropdown item
    expect(screen.queryByText('second')).toBeNull();
    fireEvent.mouseEnter(screen.getByText('first'));
    screen.getByText('second');

    fireEvent.mouseLeave(screen.getByText('first'));
    expect(screen.queryByText('second')).toBeNull();

    // Second dropdown item
    expect(screen.queryByText(/sixth/)).toBeNull();
    fireEvent.mouseEnter(screen.getByText('fifth'));
    screen.getByText(/sixth/);

    fireEvent.mouseLeave(screen.getByText('fifth'));
    expect(screen.queryByText(/sixth/)).toBeNull();
  });

  it('Two-level dropdown sub item select no secondary name', async () => {
    render(<TwoLevelDropdown {...generateProps()} />);

    fireEvent.click(screen.getByText(title));
    fireEvent.mouseEnter(screen.getByText('first'));
    fireEvent.click(screen.getByText('second'));

    expect(setSelectedItem).toHaveBeenCalledWith({ itemName: 'first', subItemName: 'second' });
  });

  it('Two-level dropdown sub item select with secondary name', async () => {
    render(<TwoLevelDropdown {...generateProps()} />);

    fireEvent.click(screen.getByText(title));
    fireEvent.mouseEnter(screen.getByText('first'));
    fireEvent.click(screen.getByText(/third/));

    expect(setSelectedItem).toHaveBeenCalledWith({
      itemName: 'first',
      subItemName: 'third',
      subItemSecondaryName: 'fourth',
    });
  });
});
