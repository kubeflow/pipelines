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

import React, { useEffect, useRef, useState } from 'react';
import { Button, Menu, MenuItem, Typography } from '@material-ui/core';

import { color, commonCss, spacing } from '../Css';
import { classes, stylesheet } from 'typestyle';
import ArrowDropDownIcon from '@material-ui/icons/ArrowDropDown';
import ArrowForwardIosIcon from '@material-ui/icons/ChevronRight';

// import NestedMenuItem from "material-ui-nested-menu-item";

export const css = stylesheet({
  defaultFont: {
    fontSize: '1rem',
  },
  dropdown: {
    // TODO: I tried finding the grey used on either "Upload pipeline" or "Filter pipelines" but couldn't. Where are these?
    border: `1px solid ${color.lightGrey}`,
    color: color.grey,
    fontWeight: 400,
    margin: 0,
  },
  dropdownMenu: {
    borderRadius: '5px',
    border: `1px solid ${color.lightGrey}`,
    margin: 0,
    padding: '5px 0',
    position: 'absolute',
    top: '100%',
    left: 0,
    minWidth: '10rem',
    cursor: 'pointer',
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
    maxWidth: '20rem',
    overflow: 'hidden',
    verticalAlign: 'top',
  },
  dropdownSubmenu: {
    left: '100%',
    top: 0,
    marginTop: '-6px',
  },
  inlineContainer: {
    display: 'inline-block',
  },
  relativeContainer: {
    position: 'relative',
  },
  active: {
    display: 'block',
  },
  hidden: {
    display: 'none',
  },
});

export interface SelectedItem {
  itemName: string;
  subItemName: string;
}

export interface DropdownItem {
  name: string;
  subItems: string[];
}

interface DropdownProps {
  title: string;
  items: DropdownItem[];
  selectedItem: SelectedItem;
  setSelectedItem: (selectedItem: SelectedItem) => void;
}

export const MultiLevelDropdown = (props: DropdownProps) => {
  const { title, items, selectedItem, setSelectedItem } = props;
  const [dropdownActive, setDropdownActive] = useState(false);
  const [subDropdown, setSubDropdown] = useState<boolean[]>(new Array(items.length).fill(false));
  const dropdownRef = useRef<HTMLDivElement>(null);
  const dropdownListRef = useRef<HTMLDivElement>(null);

  const handleDropdownClick = (event: React.MouseEvent) => {
    setDropdownActive(!dropdownActive);
    event.preventDefault();
  };

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

  return (
    <div className={classes(css.relativeContainer)}>
      <div className={classes(css.inlineContainer)} ref={dropdownRef}>
        <Button onClick={handleDropdownClick} className={classes(css.dropdown, css.textContainer)}>
          <span className={classes(css.textContainer)}>
            {selectedItem.itemName && selectedItem.subItemName ? (
              <span>
                {selectedItem.itemName}
                <ArrowForwardIosIcon
                  className={classes(css.defaultFont)}
                  key={`forwardIcon-${selectedItem.itemName}`}
                />
                {selectedItem.subItemName}
              </span>
            ) : (
              title
            )}
          </span>
          <ArrowDropDownIcon />
        </Button>
      </div>
      <div ref={dropdownListRef}>
        <ul
          className={dropdownActive ? classes(css.active, css.dropdownMenu) : classes(css.hidden)}
        >
          {items.map((item, index) => {
            return (
              <li
                className={classes(css.dropdownElement)}
                onMouseEnter={() => {
                  // Necessary if it goes from one submenu directly to another.
                  const newSubDropdown = new Array(items.length).fill(false);
                  newSubDropdown[index] = true;
                  console.log(subDropdown);
                  setSubDropdown(newSubDropdown);
                }}
                onMouseLeave={() => {
                  const newSubDropdown = [...subDropdown];
                  newSubDropdown[index] = false;
                  setSubDropdown(newSubDropdown);
                }}
                key={`index${index}`}
              >
                <div className={classes(css.textContainer, css.inlineContainer)}>{item.name}</div>
                <ArrowForwardIosIcon
                  className={classes(css.defaultFont)}
                  key={`forwardIcon${index}`}
                />
                <ul
                  className={
                    subDropdown[index]
                      ? classes(css.dropdownSubmenu, css.dropdownMenu)
                      : classes(css.hidden)
                  }
                  key={`ul${index}`}
                >
                  {item.subItems.map((subItem, subIndex) => (
                    <li
                      className={classes(css.dropdownElement, css.textContainer)}
                      key={`subIndex${subIndex}`}
                      onClick={() => {
                        setSelectedItem({ itemName: item.name, subItemName: subItem });
                        setDropdownActive(false);
                      }}
                    >
                      {subItem}
                    </li>
                  ))}
                </ul>
              </li>
            );
          })}
        </ul>
      </div>
    </div>
  );
};

export default MultiLevelDropdown;
