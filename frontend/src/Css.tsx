/*
 * Copyright 2018 The Kubeflow Authors
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
import { PaletteOptions } from '@mui/material/styles';

export const color = {
  background: '#fff',
  primary: '#1976d2',
  secondary: '#dc004e',
  success: '#4caf50',
  error: '#f44336',
  warning: '#ff9800',
  info: '#2196f3',
  textPrimary: 'rgba(0, 0, 0, 0.87)',
  textSecondary: 'rgba(0, 0, 0, 0.54)',
  divider: 'rgba(0, 0, 0, 0.12)',
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

export const fontsize = {
  small: 12,
  base: 14,
  medium: 16,
  large: 18,
  title: 18,
  pageTitle: 24,
};

export const fonts = {
  main: '"Google Sans", "Helvetica Neue", sans-serif',
  code: '"Source Code Pro", monospace',
};

declare module '@mui/material/styles' {
  interface Theme {
    fonts: typeof fonts;
  }
  interface ThemeOptions {
    fonts: typeof fonts;
  }
}

export const theme = createTheme({
  palette: {
    primary: {
      main: color.primary,
    },
    secondary: {
      main: color.secondary,
    },
    error: {
      main: color.error,
    },
    warning: {
      main: color.warning,
    },
    info: {
      main: color.info,
    },
    success: {
      main: color.success,
    },
    text: {
      primary: color.textPrimary,
      secondary: color.textSecondary,
    },
    background: {
      default: color.background,
    },
    divider: color.divider,
  },
  typography: {
    fontFamily: fonts.main,
    fontSize: 13,
    button: {
      textTransform: 'none', // Prevents auto-capitalization of button text
    },
  },
  components: {
    MuiCssBaseline: {
      styleOverrides: {
        body: {
          backgroundColor: color.background,
          color: color.textPrimary,
        },
      },
    },
  },
  fonts, // Custom property
});

export const commonCss = stylesheet({
  absoluteCenter: {
    left: 'calc(50% - 15px)',
    position: 'absolute',
    top: 'calc(50% - 15px)',
  },
  busyOverlay: {
    backgroundColor: '#ffffffaa',
    bottom: 0,
    left: 0,
    position: 'absolute',
    right: 0,
    top: 0,
    zIndex: 2,
  },
  buttonAction: {
    $nest: {
      '&:disabled': {
        backgroundColor: color.background,
      },
      '&:hover': {
        backgroundColor: theme.palette.primary.main,
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
    color: 'rgba(0, 0, 0, 0.38)',
    height: 16,
    width: 16,
  },
  link: {
    $nest: {
      '&:hover': {
        color: '#5472d3',
        textDecoration: 'underline',
        cursor: 'pointer',
      },
    },
    color: 'rgba(0, 0, 0, 0.87)',
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
  transitiveReductionSwitch: {
    position: 'absolute',
    top: 0,
    left: 0,
  },
  codeEditor: {
    $nest: {
      '& .CodeMirror': {
        height: '100%',
        width: '80%',
      },

      '& .CodeMirror-gutters': {
        backgroundColor: '#f7f7f7',
      },
    },
    background: '#f7f7f7',
    height: '100%',
  },
});

export const tailwindcss = {
  sideNavItem: 'flex flex-row flex-shrink-0',
};

export function _paddingInternal(units?: number, directions?: string): NestedCSSProperties {
  units = units || 24;
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
