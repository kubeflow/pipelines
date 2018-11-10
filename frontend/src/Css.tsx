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
import { NestedCSSProperties } from 'typestyle/lib/types';
import { style, stylesheet } from 'typestyle';

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
  inactive: '#5F6368',
  secondaryText: 'rgba(0, 0, 0, .88)',
  separator: '#e8e8e8',
  strong: '#212121',
  success: '#34a853',
  theme: '#1a73e8',
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
  tiny: 24,
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
  secondary: '"Roboto", "Helvetica Neue", sans-serif',
};

export const fontsize = {
  base: 14,
  large: 18,
  medium: 16,
  small: 12,
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
        minHeight: dimension.tiny,
        textTransform: 'none',
      },
      flatPrimary: {
        border: '1px solid #ddd',
        cursor: 'pointer',
        fontSize: fontsize.base,
        marginRight: 10,
        textTransform: 'none',
      },
      flatSecondary: {
        color: color.theme,
      },
      root: {
        color: color.theme,
        marginRight: 10,
        padding: '0 8px'
      },
    },
    MuiDialogActions: {
      root: {
        margin: 15,
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
    MuiInput: {
      input: { padding: 0 },
      root: { padding: 0 }
    },
    MuiInputAdornment: {
      positionEnd: {
        paddingRight: 0,
      },
      root: { padding: 0 },
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
  buttonAction: {
    $nest: {
      '&:disabled': {
        backgroundColor: color.background,
      },
      '&:hover': {
        backgroundColor: theme.palette.primary.dark,
      },
    },
    backgroundColor: palette.primary.main,
    color: 'white',
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
    fontSize: fontsize.large,
    fontWeight: 'bold',
    paddingBottom: 16,
    paddingTop: 20,
  },
  link: {
    $nest: {
      '&:hover': {
        color: color.theme,
        textDecoration: 'underline',
      },
    },
    color: color.strong,
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

export function _paddingInternal(units?: number, directions?: string): NestedCSSProperties {
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

export function padding(units?: number, directions?: string): string {
  return style(_paddingInternal(units, directions));
}
