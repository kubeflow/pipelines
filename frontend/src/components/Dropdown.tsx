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

import React, { useEffect, useRef, useState } from "react";
import { Button, Menu, MenuItem, Typography } from "@material-ui/core";

import { color, commonCss, spacing } from '../Css';
import { classes, stylesheet } from "typestyle";
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

export interface DropdownItem {
  name: string;
  subItems: DropdownItem[];
}

interface DropdownProps {
  title: string;
  items: DropdownItem[];
}

// Iterative array to calculate the total number of items.
function calculateItems(items: DropdownItem[]): number {
  let count = 0;
  items.forEach((item: DropdownItem) => {
    count += calculateItems(item.subItems);
    count++;
  });
  return count;
}

interface DropdownItemDisplayProps {
  items: DropdownItem[];
  subDropdown: boolean[];
  setSubDropdown: (subDropdown: boolean[]) => void;
  start?: boolean;
}

let index = -1;
const DropdownItemDisplay = (props: DropdownItemDisplayProps) => {
  const { items, subDropdown, setSubDropdown, start } = props;
  if (start) {
    index = -1;
  }
  return (
    <div>
      {
        items.map(item => {
          index++;
          const valueCopy = index;
          // console.log(item.name);
          // console.log(item.subItems);
          if (item.subItems.length === 0) {
            return (<li className={classes(css.dropdownElement)} key={valueCopy} 
            onClick={() => console.log(item.name)}>{item.name}</li>);
          }
      
          return (
            <li
              className={classes(css.dropdownElement)}
              onMouseEnter={() => {
                const newSubDropdown = [...subDropdown];
                newSubDropdown[valueCopy] = true;
                console.log(subDropdown);
                setSubDropdown(newSubDropdown);
              }}
              onMouseLeave={() => {
                const newSubDropdown = [...subDropdown];
                newSubDropdown[valueCopy] = false;
                setSubDropdown(newSubDropdown);
              }}
              key={valueCopy}
            >
              {item.name}
              <ArrowForwardIosIcon className={classes(css.defaultFont)} key={`forwardIcon${valueCopy}`} />
              <ul className={subDropdown[valueCopy] ? classes(css.dropdownSubmenu, css.dropdownMenu) : classes(css.hidden)} key={`ul${valueCopy}`}>
                <DropdownItemDisplay
                  items={item.subItems}
                  subDropdown={subDropdown}
                  setSubDropdown={setSubDropdown}
                />
              </ul>
            </li>
          );
        })
      }
    </div>
  );
}

export const Dropdown = (props: DropdownProps) => {
  const { title, items } = props;
  const [dropdownActive, setDropdownActive] = useState(false);
  const [subDropdown, setSubDropdown] = useState<boolean[]>(new Array(calculateItems(items)).fill(false));
  const dropdownRef = useRef<any>(null);

  const handleDropdownClick = (event: React.MouseEvent) => {
    setDropdownActive(!dropdownActive);
    event.preventDefault();
  };

  window.onclick = (event: Event) => {
    if (dropdownRef.current && !dropdownRef.current.contains(event.target)) {
      // TODO: The additional dropdown refs - I may not want to clear on that? We will see here.
      setDropdownActive(false);
    }
  }

  useEffect(() => {
    console.log(subDropdown);
  }, [subDropdown]);

  return (
    <div
      className={classes(css.relativeContainer)}
    >
      <div
        className={classes(css.inlineContainer)}
        ref={dropdownRef}
      >
        <Button
          onClick={handleDropdownClick}
          className={classes(css.dropdown)}
        >
          {title}
          <ArrowDropDownIcon />
        </Button>
      </div>
      <ul className={dropdownActive ? classes(css.active, css.dropdownMenu) : classes(css.hidden)}>
        <DropdownItemDisplay
          items={items}
          subDropdown={subDropdown}
          setSubDropdown={setSubDropdown}
          start
        />
      </ul>
      {/* <ul className={dropdownActive ? classes(css.active, css.dropdownMenu) : classes(css.hidden)}>
        <li
          className={classes(css.dropdownElement)}
          onMouseEnter={() => {
            const newSubDropdown = [...subDropdown];
            newSubDropdown[0] = true;
            setSubDropdown(newSubDropdown);
          }}
          onMouseLeave={() => {
            const newSubDropdown = [...subDropdown];
            newSubDropdown[0] = false;
            setSubDropdown(newSubDropdown);
          }}
        >
          Test 1 Test 1 Test 1 Test 1 Test 1 Test 1 Test 1 Test 1 Test 1 
          <ArrowForwardIosIcon className={classes(css.defaultFont)} />
          <ul className={subDropdown[0] ? classes(css.dropdownSubmenu, css.dropdownMenu) : classes(css.hidden)}>
            <li
              className={classes(css.dropdownElement)}
              onMouseEnter={() => {
                const newSubDropdown = [...subDropdown];
                newSubDropdown[1] = true;
                setSubDropdown(newSubDropdown);
              }}
              onMouseLeave={() => {
                const newSubDropdown = [...subDropdown];
                newSubDropdown[1] = false;
                setSubDropdown(newSubDropdown);
              }}
            >
              Test 1.1
              <ArrowForwardIosIcon className={classes(css.defaultFont)} />
              <ul className={subDropdown[1] ? classes(css.dropdownSubmenu, css.dropdownMenu) : classes(css.hidden)}>
                <li className={classes(css.dropdownElement)}>
                  Test 1.1.1
                </li>
                <li className={classes(css.dropdownElement)}>
                  Test 1.1.2
                </li>
              </ul>
            </li>
            <li className={classes(css.dropdownElement)}>
              Test 1.2
            </li>
          </ul>
        </li>
        <li className={classes(css.dropdownElement)}>
          Test 2
        </li>
        <li className={classes(css.dropdownElement)}>
          Test 3
        </li>
      </ul> */}
    </div>
  );
};

export default Dropdown;
