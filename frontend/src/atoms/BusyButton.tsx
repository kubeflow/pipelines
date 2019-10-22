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
import Button, { ButtonProps } from '@material-ui/core/Button';
import CircularProgress from '@material-ui/core/CircularProgress';
import { stylesheet, classes } from 'typestyle';

const css = stylesheet({
  icon: {
    height: 20,
    marginRight: 4,
    width: 20,
  },
  root: {
    cursor: 'pointer',
    marginBottom: 2, // To prevent container from flickering as the spinner shows up
    position: 'relative',
    transition: 'padding 0.3s',
  },
  rootBusy: {
    cursor: 'default',
    paddingRight: 35,
  },
  spinner: {
    opacity: 0,
    position: 'absolute',
    right: '0.8em',
    transition: 'all 0.3s',
  },
  spinnerBusy: {
    opacity: 1,
  },
});

interface BusyButtonProps extends ButtonProps {
  title: string;
  icon?: any;
  busy?: boolean;
  outlined?: boolean;
}

class BusyButton extends React.Component<BusyButtonProps> {
  public render(): JSX.Element {
    const { title, busy, className, disabled, icon, outlined, ...rest } = this.props;

    return (
      <Button
        {...rest}
        color={outlined ? 'primary' : 'secondary'}
        className={classes(css.root, busy && css.rootBusy, className)}
        disabled={busy || disabled}
      >
        {!!icon && <this.props.icon className={css.icon} />}
        <span>{title}</span>
        {busy === true && (
          <CircularProgress size={15} className={classes(css.spinner, busy && css.spinnerBusy)} />
        )}
      </Button>
    );
  }
}

export default BusyButton;
