/*
 * Copyright 2021 The Kubeflow Authors
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

import { createTheme } from '@mui/material/styles';
import { NestedCSSProperties } from 'typestyle/lib/types';
import { style, stylesheet, cssRaw } from 'typestyle';

cssRaw(`
  .LineageExplorer {
    --blue-50: #e8f0fe;
    --blue-100: #d2e3fc;
    --blue-200: #aecbfa;
    --blue-300: #8ab4f8;
    --blue-400: #669df6;
    --blue-500: #1098f7;
    --blue-600: #1098f7;
    --blue-700: #1967d2;
    --blue-800: #185abc;
    --blue-900: #174ea6;
    --red-50: #fce8e6;
    --red-700: #c5221f;
    --yellow-600: #f9ab00;
    --green-500: #34a853;
    --green-600: #1e8e3e;
    --grey-50: #f8f9fa;
    --grey-100: #f1f3f4;
    --grey-200: #e8eaed;
    --grey-300: #dadce0;
    --grey-400: #bdc1c6;
    --grey-500: #9aa0a6;
    --grey-600: #80868b;
    --grey-700: #5f6368;
    --grey-800: #3c4043;
    --grey-900: #202124;
    --orange-500: #fa7b17;
    --orange-700: #d56e0c;
    --orange-900: #b06000;
    --pink-500: #f538a0;
    --pink-700: #c92786;
    --purple-500: #a142f4;
    --card-radius: 6px;
  }
  ::-webkit-scrollbar {
    -webkit-appearance: none;
    width: 7px;
    height: 7px;
  }
  ::-webkit-scrollbar-thumb {
      border-radius: 4px;
      background-color: rgba(0,0,0,.5);
      box-shadow: 0 0 1px rgba(255,255,255,.5);
  }
`);

cssRaw(`

@font-face {
    font-family: 'PublicSans-Regular';
    src: local('PublicSans-Regular'), local('PublicSans-Regular'), url(https://cdn.jsdelivr.net/npm/public-sans@0.1.6-0/fonts/webfonts/PublicSans-Regular.woff) format('woff');
}

@font-face {
    font-family: 'PublicSans-Medium';
    src: local('PublicSans-Medium'), local('PublicSans-Medium'), url(https://cdn.jsdelivr.net/npm/public-sans@0.1.6-0/fonts/webfonts/PublicSans-Medium.woff) format('woff');
}

@font-face {
    font-family: 'PublicSans-SemiBold';
    src: local('PublicSans-SemiBold'), local('PublicSans-SemiBold'), url(https://cdn.jsdelivr.net/npm/public-sans@0.1.6-0/fonts/webfonts/PublicSans-SemiBold.woff) format('woff');
}

@font-face {
    font-family: 'PublicSans-Bold';
    src: local('PublicSans-Bold'), local('PublicSans-Bold'), url(https://cdn.jsdelivr.net/npm/public-sans@0.1.6-0/fonts/webfonts/PublicSans-Bold.woff) format('woff');
}

`);

export const color = {
  activeBg: '#eaf1fd',
  alert: '#f9ab00',
  background: '#fff',
  blue: '#4285f4',
  disabledBg: '#ddd',
  divider: '#e0e0e0',
  errorBg: '#fbe9e7',
  errorText: '#d50000',
  foreground: '#000',
  graphBg: '#f2f2f2',
  grey: '#5f6368',
  inactive: '#5f6368',
  lightGrey: '#eeeeee',
  lowContrast: '#80868b',
  secondaryText: 'rgba(0, 0, 0, .88)',
  separator: '#e8e8e8',
  strong: '#202124',
  success: '#34a853',
  successWeak: '#e6f4ea',
  terminated: '#80868b',
  theme: '#1a73e8',
  themeDarker: '#0049b5',
  warningBg: '#f9f9e1',
  warningText: '#ee8100',
  infoBg: '#f3f4ff',
  infoText: '#1a73e8',
  weak: '#9aa0a6',
  link: '#0d47a1',
  linkLight: '#5472d3',
  whiteSmoke: '#f3f3f3',
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

// tslint:disable:object-literal-sort-keys
export const zIndex = {
  DROP_ZONE_OVERLAY: 1,
  GRAPH_NODE: 1,
  BUSY_OVERLAY: 2,
  PIPELINE_SUMMARY_CARD: 2,
  SIDE_PANEL: 2,
};

export const fontsize = {
  small: 12,
  base: 14,
  medium: 16,
  large: 18,
  title: 18,
};
// tslint:enable:object-literal-sort-keys

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

export const theme = createTheme({
  palette: {
    primary: {
      main: color.theme,
      dark: color.themeDarker,
    },
    secondary: {
      main: 'rgba(0, 0, 0, .38)',
    },
    error: {
      main: color.errorText,
      light: color.errorBg,
    },
    warning: {
      main: color.warningText,
      light: color.warningBg,
    },
    info: {
      main: color.infoText,
      light: color.infoBg,
    },
    success: {
      main: color.success,
      light: color.successWeak,
    },
    grey: {
      300: color.lightGrey,
      500: color.grey,
      600: color.lowContrast,
      900: color.strong,
    },
    text: {
      primary: color.foreground,
      secondary: color.secondaryText,
    },
    background: {
      default: color.background,
      paper: color.background,
    },
    divider: color.divider,
  },
  typography: {
    fontFamily: '"Google Sans", "Helvetica Neue", sans-serif',
    fontSize: 14,
    button: {
      textTransform: 'none',
    },
  },
  components: {
    MuiButton: {
      styleOverrides: {
        root: {
          minWidth: 0,
        },
        text: {
          fontSize: 14,
          fontWeight: 'bold',
          minHeight: 32,
        },
        contained: {
          border: '1px solid #ddd',
          cursor: 'pointer',
          fontSize: 14,
          marginRight: 10,
        },
      },
    },
    MuiDialogActions: {
      styleOverrides: {
        root: {
          margin: 15,
        },
      },
    },
    MuiDialogTitle: {
      styleOverrides: {
        root: {
          fontSize: 18,
        },
      },
    },
    MuiFormControlLabel: {
      styleOverrides: {
        root: {
          marginLeft: 0,
        },
      },
    },
    MuiFormLabel: {
      styleOverrides: {
        root: {
          fontSize: 14,
          marginLeft: 5,
          marginTop: -8,
          '&.Mui-focused': {
            marginLeft: 0,
            marginTop: 0,
          },
        },
        filled: {
          marginLeft: 0,
          marginTop: 0,
        },
      },
    },
    MuiIconButton: {
      styleOverrides: {
        root: {
          padding: 9,
        },
      },
    },
    MuiInput: {
      styleOverrides: {
        root: {
          padding: 0,
        },
        input: {
          padding: 0,
        },
      },
    },
    MuiInputAdornment: {
      styleOverrides: {
        root: {
          padding: 0,
        },
        positionEnd: {
          paddingRight: 0,
        },
      },
    },
    MuiTooltip: {
      styleOverrides: {
        tooltip: {
          backgroundColor: '#666',
          color: '#f1f1f1',
          fontSize: 12,
        },
      },
    },
  },
});

export const commonCss = stylesheet({
  absoluteCenter: {
    left: 'calc(50% - 24px)',
    position: 'absolute',
    top: 'calc(50% - 24px)',
  },
  busyOverlay: {
    backgroundColor: '#ffffffaa',
    bottom: 0,
    left: 0,
    position: 'absolute',
    right: 0,
    top: 0,
    zIndex: zIndex.BUSY_OVERLAY,
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
    backgroundColor: theme.palette.primary.main,
    color: 'white',
  },
  ellipsis: {
    display: 'block',
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    whiteSpace: 'nowrap',
  },
  fit: {
    bottom: 0,
    height: '100%',
    left: 0,
    position: 'absolute',
    right: 0,
    top: 0,
    width: '100%',
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
  header2: {
    fontSize: fontsize.medium,
    fontWeight: 'bold',
    paddingBottom: 16,
    paddingTop: 20,
  },
  infoIcon: {
    color: color.lowContrast,
    height: 16,
    width: 16,
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
    background: `linear-gradient(white 30%, rgba(255,255,255,0)),
       linear-gradient(rgba(255,255,255,0), white 70%) 0 100%,
       radial-gradient(farthest-corner at 50% 0, rgba(0,0,0,.2), rgba(0,0,0,0)),
       radial-gradient(farthest-corner at 50% 100%, rgba(0,0,0,.2), rgba(0,0,0,0)) 0 100%`,
    backgroundAttachment: 'local, local, scroll, scroll',
    backgroundColor: 'white',
    backgroundRepeat: 'no-repeat',
    backgroundSize: '100% 40px, 100% 40px, 100% 2px, 100% 2px',
    overflow: 'auto',
    position: 'relative',
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
