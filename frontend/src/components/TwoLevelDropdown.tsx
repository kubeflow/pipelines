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

import React, { Ref, useRef, useState } from 'react';
import { Button, Tooltip } from '@material-ui/core';
import { color } from 'src/Css';
import { classes, stylesheet } from 'typestyle';
import ArrowDropDownIcon from '@material-ui/icons/ArrowDropDown';
import ChevronRightIcon from '@material-ui/icons/ChevronRight';

const css = stylesheet({
  defaultFont: {
    fontSize: '1rem',
  },
  dropdown: {
    border: `1px solid ${color.lightGrey}`,
    color: color.grey,
    fontWeight: 400,
    margin: 0,
  },
  dropdownMenu: {
    display: 'block',
    borderRadius: '5px',
    border: `1px solid ${color.lightGrey}`,
    margin: 0,
    padding: '5px 0',
    position: 'absolute',
    top: '100%',
    left: 0,
    minWidth: '10rem',
    cursor: 'pointer',
    backgroundColor: color.background,
  },
  dropdownElement: {
    padding: '1rem',
    position: 'relative',

    $nest: {
      '&:hover': {
        backgroundColor: `${color.lightGrey}`,
      },
    },
  },
  textContainer: {
    textOverflow: 'ellipsis',
    whiteSpace: 'nowrap',
    maxWidth: '30rem',
    overflow: 'hidden',
    verticalAlign: 'top',
  },
  dropdownSubmenu: {
    left: '100%',
    // Offsets the dropdown menu vertical padding (5px) and border (1px).
    top: '-6px',
  },
  inlineContainer: {
    display: 'inline-block',
  },
  relativeContainer: {
    position: 'relative',
  },
});

const ChevronRightSmallIcon = () => <ChevronRightIcon className={classes(css.defaultFont)} />;

export interface SelectedItem {
  itemName: string;
  subItemName: string;
  subItemSecondaryName?: string;
}

export interface DropdownSubItem {
  name: string;
  secondaryName?: string;
}

export interface DropdownItem {
  name: string;
  subItems: DropdownSubItem[];
}

interface DropdownButtonProps {
  dropdownRef: Ref<HTMLDivElement>;
  toggleDropdown: (e: React.MouseEvent) => void;
  selectedItem: SelectedItem;
  title: string;
}

function DropdownButton(props: DropdownButtonProps) {
  const { dropdownRef, toggleDropdown, selectedItem, title } = props;

  const dropdownTooltip: string =
    selectedItem.itemName && selectedItem.subItemName
      ? `${selectedItem.itemName} > ${selectedItem.subItemName}` +
        (selectedItem.subItemSecondaryName ? ` > ${selectedItem.subItemSecondaryName}` : '')
      : title;
  return (
    <div className={classes(css.inlineContainer)} ref={dropdownRef}>
      <Button onClick={toggleDropdown} className={classes(css.dropdown, css.textContainer)}>
        <Tooltip title={dropdownTooltip} enterDelay={300} placement='top-start'>
          <span className={classes(css.textContainer)}>
            {selectedItem.itemName && selectedItem.subItemName ? (
              <>
                {selectedItem.itemName}
                <ChevronRightSmallIcon />
                {selectedItem.subItemName}
                {selectedItem.subItemSecondaryName && (
                  <>
                    <ChevronRightSmallIcon />
                    {selectedItem.subItemSecondaryName}
                  </>
                )}
              </>
            ) : (
              title
            )}
          </span>
        </Tooltip>
        <ArrowDropDownIcon />
      </Button>
    </div>
  );
}

interface DropdownSubMenuProps {
  subDropdownActive: number;
  setSubDropdownActive: (subDropdownActive: number) => void;
  subDropdownIndex: number;
  item: DropdownItem;
  setSelectedItem: (selectedItem: SelectedItem) => void;
  setDropdownActive: (dropdownActive: boolean) => void;
}

function DropdownSubMenu(props: DropdownSubMenuProps) {
  const {
    subDropdownActive,
    setSubDropdownActive,
    subDropdownIndex,
    item,
    setSelectedItem,
    setDropdownActive,
  } = props;

  if (item.subItems.length === 0 || subDropdownActive !== subDropdownIndex) {
    return <></>;
  }

  return (
    <ul className={classes(css.dropdownSubmenu, css.dropdownMenu)}>
      {item.subItems.map((subItem, subIndex) => (
        <li
          className={classes(css.dropdownElement)}
          key={subIndex}
          onClick={() => {
            setSelectedItem({
              itemName: item.name,
              subItemName: subItem.name,
              subItemSecondaryName: subItem.secondaryName,
            });
            setDropdownActive(false);
            setSubDropdownActive(-1);
          }}
        >
          <Tooltip
            title={subItem.name + (subItem.secondaryName ? ` > ${subItem.secondaryName}` : '')}
            enterDelay={300}
            placement='top-start'
          >
            <span className={classes(css.textContainer, css.inlineContainer)}>
              {subItem.name}
              {subItem.secondaryName && (
                <>
                  <ChevronRightSmallIcon />
                  {subItem.secondaryName}
                </>
              )}
            </span>
          </Tooltip>
        </li>
      ))}
    </ul>
  );
}

interface DropdownMenuProps {
  dropdownListRef: Ref<HTMLDivElement>;
  dropdownActive: boolean;
  items: DropdownItem[];
  setSelectedItem: (selectedItem: SelectedItem) => void;
  setDropdownActive: (dropdownActive: boolean) => void;
}

function DropdownMenu(props: DropdownMenuProps) {
  const { dropdownListRef, dropdownActive, items, setSelectedItem, setDropdownActive } = props;

  // Provides the index of the active sub-dropdown, or '-1' if none are active.
  const [subDropdownActive, setSubDropdownActive] = useState<number>(-1);

  if (items.length === 0 || !dropdownActive) {
    return <></>;
  }

  return (
    <div ref={dropdownListRef}>
      <ul className={classes(css.dropdownMenu)}>
        {items.map((item, index) => {
          return (
            <li
              className={classes(css.dropdownElement)}
              onMouseEnter={() => setSubDropdownActive(index)}
              onMouseLeave={() => setSubDropdownActive(-1)}
              key={index}
            >
              <Tooltip title={item.name} enterDelay={300} placement='top-start'>
                <span className={classes(css.textContainer, css.inlineContainer)}>{item.name}</span>
              </Tooltip>
              <ChevronRightSmallIcon />
              <DropdownSubMenu
                subDropdownActive={subDropdownActive}
                setSubDropdownActive={setSubDropdownActive}
                subDropdownIndex={index}
                item={item}
                setSelectedItem={setSelectedItem}
                setDropdownActive={setDropdownActive}
              />
            </li>
          );
        })}
      </ul>
    </div>
  );
}

export interface TwoLevelDropdownProps {
  title: string;
  items: DropdownItem[];
  selectedItem: SelectedItem;
  setSelectedItem: (selectedItem: SelectedItem) => void;
}

function TwoLevelDropdown(props: TwoLevelDropdownProps) {
  const { title, items, selectedItem, setSelectedItem } = props;
  const [dropdownActive, setDropdownActive] = useState(false);

  // Close dropdown if the user clicks outside of the dropdown button or main menu.
  const dropdownRef = useRef<HTMLDivElement>(null);
  const dropdownListRef = useRef<HTMLDivElement>(null);
  window.onclick = (event: MouseEvent) => {
    if (
      dropdownRef.current &&
      !dropdownRef.current.contains(event.target as Node) &&
      dropdownListRef.current &&
      !dropdownListRef.current.contains(event.target as Node)
    ) {
      setDropdownActive(false);
    }
  };

  const toggleDropdown = (_: React.MouseEvent) => {
    setDropdownActive(!dropdownActive);
  };

  return (
    <div className={classes(css.relativeContainer)}>
      <DropdownButton
        dropdownRef={dropdownRef}
        toggleDropdown={toggleDropdown}
        selectedItem={selectedItem}
        title={title}
      />
      <DropdownMenu
        dropdownListRef={dropdownListRef}
        dropdownActive={dropdownActive}
        items={items}
        setSelectedItem={setSelectedItem}
        setDropdownActive={setDropdownActive}
      />
    </div>
  );
}

export default TwoLevelDropdown;
