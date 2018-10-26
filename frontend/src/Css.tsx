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

import createMuiTheme from '@material-ui/core/styles/createMuiTheme';
import { style, stylesheet } from 'typestyle';
import { NestedCSSProperties } from 'typestyle/lib/types';

export const color = {
  activeBg: '#eaf1fd',
  alert: '#f9ab00', // Google yellow 600
  background: '#fff',
  disabledBg: '#ddd',
  divider: '#e0e0e0',
  errorBg: '#FBE9E7',
  errorText: '#D50000',
  foreground: '#000',
  graphBg: '#f5f5f5',
  hoverBg: '#eee',
  inactive: '#616161',
  secondaryText: 'rgba(0, 0, 0, .66)',
  separator: '#e8e8e8',
  strong: '#000',
  success: '#34a853',
  theme: '#2979FF',
  themeDarker: '#0b59dc',
  warningBg: '#f9f9e1',
  weak: '#999',
};

export const dimension = {
  auto: 'auto',
  base: 40,
  jumbo: 64,
  large: 48,
  small: 36,
  xlarge: 56,
  xsmall: 32,
};

const baseSpacing = 24;
export const spacing = {
  base: baseSpacing,
  units: (unit: number) => baseSpacing + unit * 4,
};

export const fonts = {
  code: '"Source Code Pro", monospace',
  main: '"Google Sans", "Helvetica Neue", sans-serif',
};

export const fontsize = {
  base: 13,
  large: 16,
  medium: 14,
  small: 11,
  title: 18,
};

const palette = {
  primary: {
    dark: color.themeDarker,
    main: color.theme,
  },
  secondary: {
    main: 'rgba(0, 0, 0, .38)',
  },
};

export const theme = createMuiTheme({
  overrides: {
    MuiButton: {
      disabled: {
        backgroundColor: 'initial',
      },
      flat: {
        fontSize: fontsize.base,
        fontWeight: 'bold',
        minHeight: 24,
        padding: '4px 8px',
        textTransform: 'none',
      },
      flatPrimary: {
        backgroundColor: palette.primary.main,
        color: 'white',
      },
      flatSecondary: {
        color: color.theme,
      },
      root: {
        marginRight: 10,
      },
    },
    MuiDialogTitle: {
      root: {
        fontSize: fontsize.large,
      },
    },
    MuiFormControlLabel: {
      root: {
        marginLeft: 0,
      },
    },
    MuiFormLabel: {
      filled: {
        marginLeft: 0,
        marginTop: 0,
      },
      focused: {
        marginLeft: 0,
        marginTop: 0,
      },
      root: {
        fontSize: fontsize.base,
        marginLeft: 5,
        marginTop: -8,
      },
    },
    MuiIconButton: {
      root: {
        padding: 9,
      },
    },
    MuiTooltip: {
      tooltip: {
        backgroundColor: '#666',
        color: '#f1f1f1',
        fontSize: 12,
      }
    },
  },
  palette,
  typography: {
    fontFamily: fonts.main,
    fontSize: fontsize.base + ' !important' as any,
  },
});

export const commonCss = stylesheet({
  absoluteCenter: {
    left: 'calc(50% - 15px)',
    position: 'absolute',
    top: 'calc(50% - 15px)',
  },
  actionButton: {
    fontSize: fontsize.base,
    marginRight: 20,
    minHeight: 30,
    padding: '6px 8px',
    textTransform: 'none',
  },
  ellipsis: {
    display: 'block',
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    whiteSpace: 'nowrap',
  },
  flex: {
    alignItems: 'center !important',
    display: 'flex !important',
    flexShrink: 0,
  },
  flexColumn: {
    display: 'flex !important',
    flexDirection: 'column',
  },
  flexGrow: {
    display: 'flex !important',
    flexGrow: 1,
  },
  header: {
    color: color.strong,
    fontSize: fontsize.large,
    paddingBottom: spacing.units(-4),
    paddingTop: spacing.units(-2),
  },
  link: {
    $nest: {
      '&:hover': {
        color: color.theme,
        textDecoration: 'underline',
      },
    },
    color: 'inherit',
    cursor: 'pointer',
    textDecoration: 'none',
  },
  noShrink: {
    flexShrink: 0,
  },
  page: {
    display: 'flex',
    flexFlow: 'column',
    flexGrow: 1,
    overflow: 'auto',
  },
  pageOverflowHidden: {
    display: 'flex',
    flexFlow: 'column',
    flexGrow: 1,
    overflowX: 'auto',
    overflowY: 'hidden',
  },
  prewrap: {
    whiteSpace: 'pre-wrap',
  },
  primaryButton: {
    $nest: {
      '&:disabled': {
        backgroundColor: theme.palette.secondary.light,
      },
      '&:hover': {
        backgroundColor: theme.palette.primary.dark,
      },
    },
    backgroundColor: theme.palette.primary.main,
    color: 'white',
    fontWeight: 'lighter',
  },
  scrollContainer: {
    background:
      `linear-gradient(white 30%, rgba(255,255,255,0)),
       linear-gradient(rgba(255,255,255,0), white 70%) 0 100%,
       radial-gradient(farthest-corner at 50% 0, rgba(0,0,0,.2), rgba(0,0,0,0)),
       radial-gradient(farthest-corner at 50% 100%, rgba(0,0,0,.2), rgba(0,0,0,0)) 0 100%`,
    backgroundAttachment: 'local, local, scroll, scroll',
    backgroundColor: 'white',
    backgroundRepeat: 'no-repeat',
    backgroundSize: '100% 40px, 100% 40px, 100% 2px, 100% 2px',
    overflow: 'auto',
    paddingBottom: 20,
  },
  textField: {
    display: 'flex',
    height: 40,
    marginBottom: 20,
    marginTop: 15,
  },
  unstyled: {
    color: 'inherit',
    outline: 'none',
    textDecoration: 'none',
  },
});

export function _paddingInternal(units?: number, directions?: string) {
  units = units || baseSpacing;
  directions = directions || 'blrt';
  const rules: NestedCSSProperties = {};
  if (directions.indexOf('b') > -1) {
    rules.paddingBottom = units;
  }
  if (directions.indexOf('l') > -1) {
    rules.paddingLeft = units;
  }
  if (directions.indexOf('r') > -1) {
    rules.paddingRight = units;
  }
  if (directions.indexOf('t') > -1) {
    rules.paddingTop = units;
  }
  return rules;
}

export function padding(units?: number, directions?: string) {
  return style(_paddingInternal(units, directions));
}
