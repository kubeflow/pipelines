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

import * as Utils from '../lib/Utils';
import { statusToIcon } from './Status';
import { NodePhase } from '../lib/StatusUtils';
import { shallow } from 'enzyme';
import Tooltip from '@material-ui/core/Tooltip';

describe('Status', () => {
  // We mock this because it uses toLocaleDateString, which causes mismatches between local and CI
  // test enviroments
  const formatDateStringSpy = jest.spyOn(Utils, 'formatDateString');

  const startDate = new Date('Wed Jan 2 2019 9:10:11 GMT-0800');
  const endDate = new Date('Thu Jan 3 2019 10:11:12 GMT-0800');

  beforeEach(() => {
    formatDateStringSpy.mockImplementation((date: Date) => {
      return date === startDate ? '1/2/2019, 9:10:11 AM' : '1/3/2019, 10:11:12 AM';
    });
  });

  describe('statusToIcon', () => {
    it('handles an unknown phase', () => {
      const consoleSpy = jest.spyOn(console, 'log').mockImplementationOnce(() => null);
      const tree = shallow(statusToIcon('bad phase' as any));
      expect(tree).toMatchSnapshot();
      expect(consoleSpy).toHaveBeenLastCalledWith('Unknown node phase:', 'bad phase');
    });

    it('handles an undefined phase', () => {
      const consoleSpy = jest.spyOn(console, 'log').mockImplementationOnce(() => null);
      const tree = shallow(statusToIcon(/* no phase */));
      expect(tree).toMatchSnapshot();
      expect(consoleSpy).toHaveBeenLastCalledWith('Unknown node phase:', undefined);
    });

    it('handles ERROR phase', () => {
      const tree = shallow(statusToIcon(NodePhase.ERROR));
      expect(tree).toMatchInlineSnapshot(`
        <Tooltip
          TransitionComponent={[Function]}
          classes={
            Object {
              "popper": "MuiTooltip-popper-1",
              "popperInteractive": "MuiTooltip-popperInteractive-2",
              "tooltip": "MuiTooltip-tooltip-3",
              "tooltipPlacementBottom": "MuiTooltip-tooltipPlacementBottom-8",
              "tooltipPlacementLeft": "MuiTooltip-tooltipPlacementLeft-5",
              "tooltipPlacementRight": "MuiTooltip-tooltipPlacementRight-6",
              "tooltipPlacementTop": "MuiTooltip-tooltipPlacementTop-7",
              "touch": "MuiTooltip-touch-4",
            }
          }
          disableFocusListener={false}
          disableHoverListener={false}
          disableTouchListener={false}
          enterDelay={0}
          enterTouchDelay={1000}
          interactive={false}
          leaveDelay={0}
          leaveTouchDelay={1500}
          placement="bottom"
          theme={
            Object {
              "breakpoints": Object {
                "between": [Function],
                "down": [Function],
                "keys": Array [
                  "xs",
                  "sm",
                  "md",
                  "lg",
                  "xl",
                ],
                "only": [Function],
                "up": [Function],
                "values": Object {
                  "lg": 1280,
                  "md": 960,
                  "sm": 600,
                  "xl": 1920,
                  "xs": 0,
                },
                "width": [Function],
              },
              "direction": "ltr",
              "mixins": Object {
                "gutters": [Function],
                "toolbar": Object {
                  "@media (min-width:0px) and (orientation: landscape)": Object {
                    "minHeight": 48,
                  },
                  "@media (min-width:600px)": Object {
                    "minHeight": 64,
                  },
                  "minHeight": 56,
                },
              },
              "overrides": Object {},
              "palette": Object {
                "action": Object {
                  "active": "rgba(0, 0, 0, 0.54)",
                  "disabled": "rgba(0, 0, 0, 0.26)",
                  "disabledBackground": "rgba(0, 0, 0, 0.12)",
                  "hover": "rgba(0, 0, 0, 0.08)",
                  "hoverOpacity": 0.08,
                  "selected": "rgba(0, 0, 0, 0.14)",
                },
                "augmentColor": [Function],
                "background": Object {
                  "default": "#fafafa",
                  "paper": "#fff",
                },
                "common": Object {
                  "black": "#000",
                  "white": "#fff",
                },
                "contrastThreshold": 3,
                "divider": "rgba(0, 0, 0, 0.12)",
                "error": Object {
                  "contrastText": "#fff",
                  "dark": "#d32f2f",
                  "light": "#e57373",
                  "main": "#f44336",
                },
                "getContrastText": [Function],
                "grey": Object {
                  "100": "#f5f5f5",
                  "200": "#eeeeee",
                  "300": "#e0e0e0",
                  "400": "#bdbdbd",
                  "50": "#fafafa",
                  "500": "#9e9e9e",
                  "600": "#757575",
                  "700": "#616161",
                  "800": "#424242",
                  "900": "#212121",
                  "A100": "#d5d5d5",
                  "A200": "#aaaaaa",
                  "A400": "#303030",
                  "A700": "#616161",
                },
                "primary": Object {
                  "contrastText": "#fff",
                  "dark": "#303f9f",
                  "light": "#7986cb",
                  "main": "#3f51b5",
                },
                "secondary": Object {
                  "contrastText": "#fff",
                  "dark": "#c51162",
                  "light": "#ff4081",
                  "main": "#f50057",
                },
                "text": Object {
                  "disabled": "rgba(0, 0, 0, 0.38)",
                  "hint": "rgba(0, 0, 0, 0.38)",
                  "primary": "rgba(0, 0, 0, 0.87)",
                  "secondary": "rgba(0, 0, 0, 0.54)",
                },
                "tonalOffset": 0.2,
                "type": "light",
              },
              "props": Object {},
              "shadows": Array [
                "none",
                "0px 1px 3px 0px rgba(0,0,0,0.2),0px 1px 1px 0px rgba(0,0,0,0.14),0px 2px 1px -1px rgba(0,0,0,0.12)",
                "0px 1px 5px 0px rgba(0,0,0,0.2),0px 2px 2px 0px rgba(0,0,0,0.14),0px 3px 1px -2px rgba(0,0,0,0.12)",
                "0px 1px 8px 0px rgba(0,0,0,0.2),0px 3px 4px 0px rgba(0,0,0,0.14),0px 3px 3px -2px rgba(0,0,0,0.12)",
                "0px 2px 4px -1px rgba(0,0,0,0.2),0px 4px 5px 0px rgba(0,0,0,0.14),0px 1px 10px 0px rgba(0,0,0,0.12)",
                "0px 3px 5px -1px rgba(0,0,0,0.2),0px 5px 8px 0px rgba(0,0,0,0.14),0px 1px 14px 0px rgba(0,0,0,0.12)",
                "0px 3px 5px -1px rgba(0,0,0,0.2),0px 6px 10px 0px rgba(0,0,0,0.14),0px 1px 18px 0px rgba(0,0,0,0.12)",
                "0px 4px 5px -2px rgba(0,0,0,0.2),0px 7px 10px 1px rgba(0,0,0,0.14),0px 2px 16px 1px rgba(0,0,0,0.12)",
                "0px 5px 5px -3px rgba(0,0,0,0.2),0px 8px 10px 1px rgba(0,0,0,0.14),0px 3px 14px 2px rgba(0,0,0,0.12)",
                "0px 5px 6px -3px rgba(0,0,0,0.2),0px 9px 12px 1px rgba(0,0,0,0.14),0px 3px 16px 2px rgba(0,0,0,0.12)",
                "0px 6px 6px -3px rgba(0,0,0,0.2),0px 10px 14px 1px rgba(0,0,0,0.14),0px 4px 18px 3px rgba(0,0,0,0.12)",
                "0px 6px 7px -4px rgba(0,0,0,0.2),0px 11px 15px 1px rgba(0,0,0,0.14),0px 4px 20px 3px rgba(0,0,0,0.12)",
                "0px 7px 8px -4px rgba(0,0,0,0.2),0px 12px 17px 2px rgba(0,0,0,0.14),0px 5px 22px 4px rgba(0,0,0,0.12)",
                "0px 7px 8px -4px rgba(0,0,0,0.2),0px 13px 19px 2px rgba(0,0,0,0.14),0px 5px 24px 4px rgba(0,0,0,0.12)",
                "0px 7px 9px -4px rgba(0,0,0,0.2),0px 14px 21px 2px rgba(0,0,0,0.14),0px 5px 26px 4px rgba(0,0,0,0.12)",
                "0px 8px 9px -5px rgba(0,0,0,0.2),0px 15px 22px 2px rgba(0,0,0,0.14),0px 6px 28px 5px rgba(0,0,0,0.12)",
                "0px 8px 10px -5px rgba(0,0,0,0.2),0px 16px 24px 2px rgba(0,0,0,0.14),0px 6px 30px 5px rgba(0,0,0,0.12)",
                "0px 8px 11px -5px rgba(0,0,0,0.2),0px 17px 26px 2px rgba(0,0,0,0.14),0px 6px 32px 5px rgba(0,0,0,0.12)",
                "0px 9px 11px -5px rgba(0,0,0,0.2),0px 18px 28px 2px rgba(0,0,0,0.14),0px 7px 34px 6px rgba(0,0,0,0.12)",
                "0px 9px 12px -6px rgba(0,0,0,0.2),0px 19px 29px 2px rgba(0,0,0,0.14),0px 7px 36px 6px rgba(0,0,0,0.12)",
                "0px 10px 13px -6px rgba(0,0,0,0.2),0px 20px 31px 3px rgba(0,0,0,0.14),0px 8px 38px 7px rgba(0,0,0,0.12)",
                "0px 10px 13px -6px rgba(0,0,0,0.2),0px 21px 33px 3px rgba(0,0,0,0.14),0px 8px 40px 7px rgba(0,0,0,0.12)",
                "0px 10px 14px -6px rgba(0,0,0,0.2),0px 22px 35px 3px rgba(0,0,0,0.14),0px 8px 42px 7px rgba(0,0,0,0.12)",
                "0px 11px 14px -7px rgba(0,0,0,0.2),0px 23px 36px 3px rgba(0,0,0,0.14),0px 9px 44px 8px rgba(0,0,0,0.12)",
                "0px 11px 15px -7px rgba(0,0,0,0.2),0px 24px 38px 3px rgba(0,0,0,0.14),0px 9px 46px 8px rgba(0,0,0,0.12)",
              ],
              "shape": Object {
                "borderRadius": 4,
              },
              "spacing": Object {
                "unit": 8,
              },
              "transitions": Object {
                "create": [Function],
                "duration": Object {
                  "complex": 375,
                  "enteringScreen": 225,
                  "leavingScreen": 195,
                  "short": 250,
                  "shorter": 200,
                  "shortest": 150,
                  "standard": 300,
                },
                "easing": Object {
                  "easeIn": "cubic-bezier(0.4, 0, 1, 1)",
                  "easeInOut": "cubic-bezier(0.4, 0, 0.2, 1)",
                  "easeOut": "cubic-bezier(0.0, 0, 0.2, 1)",
                  "sharp": "cubic-bezier(0.4, 0, 0.6, 1)",
                },
                "getAutoHeightDuration": [Function],
              },
              "typography": Object {
                "body1": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.875rem",
                  "fontWeight": 400,
                  "lineHeight": "1.46429em",
                },
                "body1Next": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1rem",
                  "fontWeight": 400,
                  "letterSpacing": "0.00938em",
                  "lineHeight": 1.5,
                },
                "body2": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.875rem",
                  "fontWeight": 500,
                  "lineHeight": "1.71429em",
                },
                "body2Next": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.875rem",
                  "fontWeight": 400,
                  "letterSpacing": "0.01071em",
                  "lineHeight": 1.5,
                },
                "button": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.875rem",
                  "fontWeight": 500,
                  "textTransform": "uppercase",
                },
                "buttonNext": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.875rem",
                  "fontWeight": 500,
                  "letterSpacing": "0.02857em",
                  "lineHeight": 1.5,
                  "textTransform": "uppercase",
                },
                "caption": Object {
                  "color": "rgba(0, 0, 0, 0.54)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.75rem",
                  "fontWeight": 400,
                  "lineHeight": "1.375em",
                },
                "captionNext": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.75rem",
                  "fontWeight": 400,
                  "letterSpacing": "0.03333em",
                  "lineHeight": 1.66,
                },
                "display1": Object {
                  "color": "rgba(0, 0, 0, 0.54)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "2.125rem",
                  "fontWeight": 400,
                  "lineHeight": "1.20588em",
                },
                "display2": Object {
                  "color": "rgba(0, 0, 0, 0.54)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "2.8125rem",
                  "fontWeight": 400,
                  "lineHeight": "1.13333em",
                  "marginLeft": "-.02em",
                },
                "display3": Object {
                  "color": "rgba(0, 0, 0, 0.54)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "3.5rem",
                  "fontWeight": 400,
                  "letterSpacing": "-.02em",
                  "lineHeight": "1.30357em",
                  "marginLeft": "-.02em",
                },
                "display4": Object {
                  "color": "rgba(0, 0, 0, 0.54)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "7rem",
                  "fontWeight": 300,
                  "letterSpacing": "-.04em",
                  "lineHeight": "1.14286em",
                  "marginLeft": "-.04em",
                },
                "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                "fontSize": 14,
                "fontWeightLight": 300,
                "fontWeightMedium": 500,
                "fontWeightRegular": 400,
                "h1": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "6rem",
                  "fontWeight": 300,
                  "letterSpacing": "-0.01562em",
                  "lineHeight": 1,
                },
                "h2": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "3.75rem",
                  "fontWeight": 300,
                  "letterSpacing": "-0.00833em",
                  "lineHeight": 1,
                },
                "h3": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "3rem",
                  "fontWeight": 400,
                  "letterSpacing": "0em",
                  "lineHeight": 1.04,
                },
                "h4": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "2.125rem",
                  "fontWeight": 400,
                  "letterSpacing": "0.00735em",
                  "lineHeight": 1.17,
                },
                "h5": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1.5rem",
                  "fontWeight": 400,
                  "letterSpacing": "0em",
                  "lineHeight": 1.33,
                },
                "h6": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1.25rem",
                  "fontWeight": 500,
                  "letterSpacing": "0.0075em",
                  "lineHeight": 1.6,
                },
                "headline": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1.5rem",
                  "fontWeight": 400,
                  "lineHeight": "1.35417em",
                },
                "overline": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.75rem",
                  "fontWeight": 400,
                  "letterSpacing": "0.08333em",
                  "lineHeight": 2.66,
                  "textTransform": "uppercase",
                },
                "pxToRem": [Function],
                "round": [Function],
                "subheading": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1rem",
                  "fontWeight": 400,
                  "lineHeight": "1.5em",
                },
                "subtitle1": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1rem",
                  "fontWeight": 400,
                  "letterSpacing": "0.00938em",
                  "lineHeight": 1.75,
                },
                "subtitle2": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.875rem",
                  "fontWeight": 500,
                  "letterSpacing": "0.00714em",
                  "lineHeight": 1.57,
                },
                "title": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1.3125rem",
                  "fontWeight": 500,
                  "lineHeight": "1.16667em",
                },
                "useNextVariants": false,
              },
              "zIndex": Object {
                "appBar": 1100,
                "drawer": 1200,
                "mobileStepper": 1000,
                "modal": 1300,
                "snackbar": 1400,
                "tooltip": 1500,
              },
            }
          }
          title={
            <div>
              <div>
                Error while running this resource
              </div>
            </div>
          }
        >
          <span
            style={
              Object {
                "height": 18,
              }
            }
          >
            <pure(ErrorIcon)
              style={
                Object {
                  "color": "#d50000",
                  "height": 18,
                  "width": 18,
                }
              }
            />
          </span>
        </Tooltip>
      `);
    });

    it('handles FAILED phase', () => {
      const tree = shallow(statusToIcon(NodePhase.FAILED));
      expect(tree).toMatchInlineSnapshot(`
        <Tooltip
          TransitionComponent={[Function]}
          classes={
            Object {
              "popper": "MuiTooltip-popper-1",
              "popperInteractive": "MuiTooltip-popperInteractive-2",
              "tooltip": "MuiTooltip-tooltip-3",
              "tooltipPlacementBottom": "MuiTooltip-tooltipPlacementBottom-8",
              "tooltipPlacementLeft": "MuiTooltip-tooltipPlacementLeft-5",
              "tooltipPlacementRight": "MuiTooltip-tooltipPlacementRight-6",
              "tooltipPlacementTop": "MuiTooltip-tooltipPlacementTop-7",
              "touch": "MuiTooltip-touch-4",
            }
          }
          disableFocusListener={false}
          disableHoverListener={false}
          disableTouchListener={false}
          enterDelay={0}
          enterTouchDelay={1000}
          interactive={false}
          leaveDelay={0}
          leaveTouchDelay={1500}
          placement="bottom"
          theme={
            Object {
              "breakpoints": Object {
                "between": [Function],
                "down": [Function],
                "keys": Array [
                  "xs",
                  "sm",
                  "md",
                  "lg",
                  "xl",
                ],
                "only": [Function],
                "up": [Function],
                "values": Object {
                  "lg": 1280,
                  "md": 960,
                  "sm": 600,
                  "xl": 1920,
                  "xs": 0,
                },
                "width": [Function],
              },
              "direction": "ltr",
              "mixins": Object {
                "gutters": [Function],
                "toolbar": Object {
                  "@media (min-width:0px) and (orientation: landscape)": Object {
                    "minHeight": 48,
                  },
                  "@media (min-width:600px)": Object {
                    "minHeight": 64,
                  },
                  "minHeight": 56,
                },
              },
              "overrides": Object {},
              "palette": Object {
                "action": Object {
                  "active": "rgba(0, 0, 0, 0.54)",
                  "disabled": "rgba(0, 0, 0, 0.26)",
                  "disabledBackground": "rgba(0, 0, 0, 0.12)",
                  "hover": "rgba(0, 0, 0, 0.08)",
                  "hoverOpacity": 0.08,
                  "selected": "rgba(0, 0, 0, 0.14)",
                },
                "augmentColor": [Function],
                "background": Object {
                  "default": "#fafafa",
                  "paper": "#fff",
                },
                "common": Object {
                  "black": "#000",
                  "white": "#fff",
                },
                "contrastThreshold": 3,
                "divider": "rgba(0, 0, 0, 0.12)",
                "error": Object {
                  "contrastText": "#fff",
                  "dark": "#d32f2f",
                  "light": "#e57373",
                  "main": "#f44336",
                },
                "getContrastText": [Function],
                "grey": Object {
                  "100": "#f5f5f5",
                  "200": "#eeeeee",
                  "300": "#e0e0e0",
                  "400": "#bdbdbd",
                  "50": "#fafafa",
                  "500": "#9e9e9e",
                  "600": "#757575",
                  "700": "#616161",
                  "800": "#424242",
                  "900": "#212121",
                  "A100": "#d5d5d5",
                  "A200": "#aaaaaa",
                  "A400": "#303030",
                  "A700": "#616161",
                },
                "primary": Object {
                  "contrastText": "#fff",
                  "dark": "#303f9f",
                  "light": "#7986cb",
                  "main": "#3f51b5",
                },
                "secondary": Object {
                  "contrastText": "#fff",
                  "dark": "#c51162",
                  "light": "#ff4081",
                  "main": "#f50057",
                },
                "text": Object {
                  "disabled": "rgba(0, 0, 0, 0.38)",
                  "hint": "rgba(0, 0, 0, 0.38)",
                  "primary": "rgba(0, 0, 0, 0.87)",
                  "secondary": "rgba(0, 0, 0, 0.54)",
                },
                "tonalOffset": 0.2,
                "type": "light",
              },
              "props": Object {},
              "shadows": Array [
                "none",
                "0px 1px 3px 0px rgba(0,0,0,0.2),0px 1px 1px 0px rgba(0,0,0,0.14),0px 2px 1px -1px rgba(0,0,0,0.12)",
                "0px 1px 5px 0px rgba(0,0,0,0.2),0px 2px 2px 0px rgba(0,0,0,0.14),0px 3px 1px -2px rgba(0,0,0,0.12)",
                "0px 1px 8px 0px rgba(0,0,0,0.2),0px 3px 4px 0px rgba(0,0,0,0.14),0px 3px 3px -2px rgba(0,0,0,0.12)",
                "0px 2px 4px -1px rgba(0,0,0,0.2),0px 4px 5px 0px rgba(0,0,0,0.14),0px 1px 10px 0px rgba(0,0,0,0.12)",
                "0px 3px 5px -1px rgba(0,0,0,0.2),0px 5px 8px 0px rgba(0,0,0,0.14),0px 1px 14px 0px rgba(0,0,0,0.12)",
                "0px 3px 5px -1px rgba(0,0,0,0.2),0px 6px 10px 0px rgba(0,0,0,0.14),0px 1px 18px 0px rgba(0,0,0,0.12)",
                "0px 4px 5px -2px rgba(0,0,0,0.2),0px 7px 10px 1px rgba(0,0,0,0.14),0px 2px 16px 1px rgba(0,0,0,0.12)",
                "0px 5px 5px -3px rgba(0,0,0,0.2),0px 8px 10px 1px rgba(0,0,0,0.14),0px 3px 14px 2px rgba(0,0,0,0.12)",
                "0px 5px 6px -3px rgba(0,0,0,0.2),0px 9px 12px 1px rgba(0,0,0,0.14),0px 3px 16px 2px rgba(0,0,0,0.12)",
                "0px 6px 6px -3px rgba(0,0,0,0.2),0px 10px 14px 1px rgba(0,0,0,0.14),0px 4px 18px 3px rgba(0,0,0,0.12)",
                "0px 6px 7px -4px rgba(0,0,0,0.2),0px 11px 15px 1px rgba(0,0,0,0.14),0px 4px 20px 3px rgba(0,0,0,0.12)",
                "0px 7px 8px -4px rgba(0,0,0,0.2),0px 12px 17px 2px rgba(0,0,0,0.14),0px 5px 22px 4px rgba(0,0,0,0.12)",
                "0px 7px 8px -4px rgba(0,0,0,0.2),0px 13px 19px 2px rgba(0,0,0,0.14),0px 5px 24px 4px rgba(0,0,0,0.12)",
                "0px 7px 9px -4px rgba(0,0,0,0.2),0px 14px 21px 2px rgba(0,0,0,0.14),0px 5px 26px 4px rgba(0,0,0,0.12)",
                "0px 8px 9px -5px rgba(0,0,0,0.2),0px 15px 22px 2px rgba(0,0,0,0.14),0px 6px 28px 5px rgba(0,0,0,0.12)",
                "0px 8px 10px -5px rgba(0,0,0,0.2),0px 16px 24px 2px rgba(0,0,0,0.14),0px 6px 30px 5px rgba(0,0,0,0.12)",
                "0px 8px 11px -5px rgba(0,0,0,0.2),0px 17px 26px 2px rgba(0,0,0,0.14),0px 6px 32px 5px rgba(0,0,0,0.12)",
                "0px 9px 11px -5px rgba(0,0,0,0.2),0px 18px 28px 2px rgba(0,0,0,0.14),0px 7px 34px 6px rgba(0,0,0,0.12)",
                "0px 9px 12px -6px rgba(0,0,0,0.2),0px 19px 29px 2px rgba(0,0,0,0.14),0px 7px 36px 6px rgba(0,0,0,0.12)",
                "0px 10px 13px -6px rgba(0,0,0,0.2),0px 20px 31px 3px rgba(0,0,0,0.14),0px 8px 38px 7px rgba(0,0,0,0.12)",
                "0px 10px 13px -6px rgba(0,0,0,0.2),0px 21px 33px 3px rgba(0,0,0,0.14),0px 8px 40px 7px rgba(0,0,0,0.12)",
                "0px 10px 14px -6px rgba(0,0,0,0.2),0px 22px 35px 3px rgba(0,0,0,0.14),0px 8px 42px 7px rgba(0,0,0,0.12)",
                "0px 11px 14px -7px rgba(0,0,0,0.2),0px 23px 36px 3px rgba(0,0,0,0.14),0px 9px 44px 8px rgba(0,0,0,0.12)",
                "0px 11px 15px -7px rgba(0,0,0,0.2),0px 24px 38px 3px rgba(0,0,0,0.14),0px 9px 46px 8px rgba(0,0,0,0.12)",
              ],
              "shape": Object {
                "borderRadius": 4,
              },
              "spacing": Object {
                "unit": 8,
              },
              "transitions": Object {
                "create": [Function],
                "duration": Object {
                  "complex": 375,
                  "enteringScreen": 225,
                  "leavingScreen": 195,
                  "short": 250,
                  "shorter": 200,
                  "shortest": 150,
                  "standard": 300,
                },
                "easing": Object {
                  "easeIn": "cubic-bezier(0.4, 0, 1, 1)",
                  "easeInOut": "cubic-bezier(0.4, 0, 0.2, 1)",
                  "easeOut": "cubic-bezier(0.0, 0, 0.2, 1)",
                  "sharp": "cubic-bezier(0.4, 0, 0.6, 1)",
                },
                "getAutoHeightDuration": [Function],
              },
              "typography": Object {
                "body1": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.875rem",
                  "fontWeight": 400,
                  "lineHeight": "1.46429em",
                },
                "body1Next": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1rem",
                  "fontWeight": 400,
                  "letterSpacing": "0.00938em",
                  "lineHeight": 1.5,
                },
                "body2": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.875rem",
                  "fontWeight": 500,
                  "lineHeight": "1.71429em",
                },
                "body2Next": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.875rem",
                  "fontWeight": 400,
                  "letterSpacing": "0.01071em",
                  "lineHeight": 1.5,
                },
                "button": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.875rem",
                  "fontWeight": 500,
                  "textTransform": "uppercase",
                },
                "buttonNext": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.875rem",
                  "fontWeight": 500,
                  "letterSpacing": "0.02857em",
                  "lineHeight": 1.5,
                  "textTransform": "uppercase",
                },
                "caption": Object {
                  "color": "rgba(0, 0, 0, 0.54)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.75rem",
                  "fontWeight": 400,
                  "lineHeight": "1.375em",
                },
                "captionNext": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.75rem",
                  "fontWeight": 400,
                  "letterSpacing": "0.03333em",
                  "lineHeight": 1.66,
                },
                "display1": Object {
                  "color": "rgba(0, 0, 0, 0.54)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "2.125rem",
                  "fontWeight": 400,
                  "lineHeight": "1.20588em",
                },
                "display2": Object {
                  "color": "rgba(0, 0, 0, 0.54)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "2.8125rem",
                  "fontWeight": 400,
                  "lineHeight": "1.13333em",
                  "marginLeft": "-.02em",
                },
                "display3": Object {
                  "color": "rgba(0, 0, 0, 0.54)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "3.5rem",
                  "fontWeight": 400,
                  "letterSpacing": "-.02em",
                  "lineHeight": "1.30357em",
                  "marginLeft": "-.02em",
                },
                "display4": Object {
                  "color": "rgba(0, 0, 0, 0.54)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "7rem",
                  "fontWeight": 300,
                  "letterSpacing": "-.04em",
                  "lineHeight": "1.14286em",
                  "marginLeft": "-.04em",
                },
                "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                "fontSize": 14,
                "fontWeightLight": 300,
                "fontWeightMedium": 500,
                "fontWeightRegular": 400,
                "h1": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "6rem",
                  "fontWeight": 300,
                  "letterSpacing": "-0.01562em",
                  "lineHeight": 1,
                },
                "h2": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "3.75rem",
                  "fontWeight": 300,
                  "letterSpacing": "-0.00833em",
                  "lineHeight": 1,
                },
                "h3": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "3rem",
                  "fontWeight": 400,
                  "letterSpacing": "0em",
                  "lineHeight": 1.04,
                },
                "h4": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "2.125rem",
                  "fontWeight": 400,
                  "letterSpacing": "0.00735em",
                  "lineHeight": 1.17,
                },
                "h5": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1.5rem",
                  "fontWeight": 400,
                  "letterSpacing": "0em",
                  "lineHeight": 1.33,
                },
                "h6": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1.25rem",
                  "fontWeight": 500,
                  "letterSpacing": "0.0075em",
                  "lineHeight": 1.6,
                },
                "headline": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1.5rem",
                  "fontWeight": 400,
                  "lineHeight": "1.35417em",
                },
                "overline": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.75rem",
                  "fontWeight": 400,
                  "letterSpacing": "0.08333em",
                  "lineHeight": 2.66,
                  "textTransform": "uppercase",
                },
                "pxToRem": [Function],
                "round": [Function],
                "subheading": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1rem",
                  "fontWeight": 400,
                  "lineHeight": "1.5em",
                },
                "subtitle1": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1rem",
                  "fontWeight": 400,
                  "letterSpacing": "0.00938em",
                  "lineHeight": 1.75,
                },
                "subtitle2": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.875rem",
                  "fontWeight": 500,
                  "letterSpacing": "0.00714em",
                  "lineHeight": 1.57,
                },
                "title": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1.3125rem",
                  "fontWeight": 500,
                  "lineHeight": "1.16667em",
                },
                "useNextVariants": false,
              },
              "zIndex": Object {
                "appBar": 1100,
                "drawer": 1200,
                "mobileStepper": 1000,
                "modal": 1300,
                "snackbar": 1400,
                "tooltip": 1500,
              },
            }
          }
          title={
            <div>
              <div>
                Resource failed to execute
              </div>
            </div>
          }
        >
          <span
            style={
              Object {
                "height": 18,
              }
            }
          >
            <pure(ErrorIcon)
              style={
                Object {
                  "color": "#d50000",
                  "height": 18,
                  "width": 18,
                }
              }
            />
          </span>
        </Tooltip>
      `);
    });

    it('handles PENDING phase', () => {
      const tree = shallow(statusToIcon(NodePhase.PENDING));
      expect(tree).toMatchInlineSnapshot(`
        <Tooltip
          TransitionComponent={[Function]}
          classes={
            Object {
              "popper": "MuiTooltip-popper-1",
              "popperInteractive": "MuiTooltip-popperInteractive-2",
              "tooltip": "MuiTooltip-tooltip-3",
              "tooltipPlacementBottom": "MuiTooltip-tooltipPlacementBottom-8",
              "tooltipPlacementLeft": "MuiTooltip-tooltipPlacementLeft-5",
              "tooltipPlacementRight": "MuiTooltip-tooltipPlacementRight-6",
              "tooltipPlacementTop": "MuiTooltip-tooltipPlacementTop-7",
              "touch": "MuiTooltip-touch-4",
            }
          }
          disableFocusListener={false}
          disableHoverListener={false}
          disableTouchListener={false}
          enterDelay={0}
          enterTouchDelay={1000}
          interactive={false}
          leaveDelay={0}
          leaveTouchDelay={1500}
          placement="bottom"
          theme={
            Object {
              "breakpoints": Object {
                "between": [Function],
                "down": [Function],
                "keys": Array [
                  "xs",
                  "sm",
                  "md",
                  "lg",
                  "xl",
                ],
                "only": [Function],
                "up": [Function],
                "values": Object {
                  "lg": 1280,
                  "md": 960,
                  "sm": 600,
                  "xl": 1920,
                  "xs": 0,
                },
                "width": [Function],
              },
              "direction": "ltr",
              "mixins": Object {
                "gutters": [Function],
                "toolbar": Object {
                  "@media (min-width:0px) and (orientation: landscape)": Object {
                    "minHeight": 48,
                  },
                  "@media (min-width:600px)": Object {
                    "minHeight": 64,
                  },
                  "minHeight": 56,
                },
              },
              "overrides": Object {},
              "palette": Object {
                "action": Object {
                  "active": "rgba(0, 0, 0, 0.54)",
                  "disabled": "rgba(0, 0, 0, 0.26)",
                  "disabledBackground": "rgba(0, 0, 0, 0.12)",
                  "hover": "rgba(0, 0, 0, 0.08)",
                  "hoverOpacity": 0.08,
                  "selected": "rgba(0, 0, 0, 0.14)",
                },
                "augmentColor": [Function],
                "background": Object {
                  "default": "#fafafa",
                  "paper": "#fff",
                },
                "common": Object {
                  "black": "#000",
                  "white": "#fff",
                },
                "contrastThreshold": 3,
                "divider": "rgba(0, 0, 0, 0.12)",
                "error": Object {
                  "contrastText": "#fff",
                  "dark": "#d32f2f",
                  "light": "#e57373",
                  "main": "#f44336",
                },
                "getContrastText": [Function],
                "grey": Object {
                  "100": "#f5f5f5",
                  "200": "#eeeeee",
                  "300": "#e0e0e0",
                  "400": "#bdbdbd",
                  "50": "#fafafa",
                  "500": "#9e9e9e",
                  "600": "#757575",
                  "700": "#616161",
                  "800": "#424242",
                  "900": "#212121",
                  "A100": "#d5d5d5",
                  "A200": "#aaaaaa",
                  "A400": "#303030",
                  "A700": "#616161",
                },
                "primary": Object {
                  "contrastText": "#fff",
                  "dark": "#303f9f",
                  "light": "#7986cb",
                  "main": "#3f51b5",
                },
                "secondary": Object {
                  "contrastText": "#fff",
                  "dark": "#c51162",
                  "light": "#ff4081",
                  "main": "#f50057",
                },
                "text": Object {
                  "disabled": "rgba(0, 0, 0, 0.38)",
                  "hint": "rgba(0, 0, 0, 0.38)",
                  "primary": "rgba(0, 0, 0, 0.87)",
                  "secondary": "rgba(0, 0, 0, 0.54)",
                },
                "tonalOffset": 0.2,
                "type": "light",
              },
              "props": Object {},
              "shadows": Array [
                "none",
                "0px 1px 3px 0px rgba(0,0,0,0.2),0px 1px 1px 0px rgba(0,0,0,0.14),0px 2px 1px -1px rgba(0,0,0,0.12)",
                "0px 1px 5px 0px rgba(0,0,0,0.2),0px 2px 2px 0px rgba(0,0,0,0.14),0px 3px 1px -2px rgba(0,0,0,0.12)",
                "0px 1px 8px 0px rgba(0,0,0,0.2),0px 3px 4px 0px rgba(0,0,0,0.14),0px 3px 3px -2px rgba(0,0,0,0.12)",
                "0px 2px 4px -1px rgba(0,0,0,0.2),0px 4px 5px 0px rgba(0,0,0,0.14),0px 1px 10px 0px rgba(0,0,0,0.12)",
                "0px 3px 5px -1px rgba(0,0,0,0.2),0px 5px 8px 0px rgba(0,0,0,0.14),0px 1px 14px 0px rgba(0,0,0,0.12)",
                "0px 3px 5px -1px rgba(0,0,0,0.2),0px 6px 10px 0px rgba(0,0,0,0.14),0px 1px 18px 0px rgba(0,0,0,0.12)",
                "0px 4px 5px -2px rgba(0,0,0,0.2),0px 7px 10px 1px rgba(0,0,0,0.14),0px 2px 16px 1px rgba(0,0,0,0.12)",
                "0px 5px 5px -3px rgba(0,0,0,0.2),0px 8px 10px 1px rgba(0,0,0,0.14),0px 3px 14px 2px rgba(0,0,0,0.12)",
                "0px 5px 6px -3px rgba(0,0,0,0.2),0px 9px 12px 1px rgba(0,0,0,0.14),0px 3px 16px 2px rgba(0,0,0,0.12)",
                "0px 6px 6px -3px rgba(0,0,0,0.2),0px 10px 14px 1px rgba(0,0,0,0.14),0px 4px 18px 3px rgba(0,0,0,0.12)",
                "0px 6px 7px -4px rgba(0,0,0,0.2),0px 11px 15px 1px rgba(0,0,0,0.14),0px 4px 20px 3px rgba(0,0,0,0.12)",
                "0px 7px 8px -4px rgba(0,0,0,0.2),0px 12px 17px 2px rgba(0,0,0,0.14),0px 5px 22px 4px rgba(0,0,0,0.12)",
                "0px 7px 8px -4px rgba(0,0,0,0.2),0px 13px 19px 2px rgba(0,0,0,0.14),0px 5px 24px 4px rgba(0,0,0,0.12)",
                "0px 7px 9px -4px rgba(0,0,0,0.2),0px 14px 21px 2px rgba(0,0,0,0.14),0px 5px 26px 4px rgba(0,0,0,0.12)",
                "0px 8px 9px -5px rgba(0,0,0,0.2),0px 15px 22px 2px rgba(0,0,0,0.14),0px 6px 28px 5px rgba(0,0,0,0.12)",
                "0px 8px 10px -5px rgba(0,0,0,0.2),0px 16px 24px 2px rgba(0,0,0,0.14),0px 6px 30px 5px rgba(0,0,0,0.12)",
                "0px 8px 11px -5px rgba(0,0,0,0.2),0px 17px 26px 2px rgba(0,0,0,0.14),0px 6px 32px 5px rgba(0,0,0,0.12)",
                "0px 9px 11px -5px rgba(0,0,0,0.2),0px 18px 28px 2px rgba(0,0,0,0.14),0px 7px 34px 6px rgba(0,0,0,0.12)",
                "0px 9px 12px -6px rgba(0,0,0,0.2),0px 19px 29px 2px rgba(0,0,0,0.14),0px 7px 36px 6px rgba(0,0,0,0.12)",
                "0px 10px 13px -6px rgba(0,0,0,0.2),0px 20px 31px 3px rgba(0,0,0,0.14),0px 8px 38px 7px rgba(0,0,0,0.12)",
                "0px 10px 13px -6px rgba(0,0,0,0.2),0px 21px 33px 3px rgba(0,0,0,0.14),0px 8px 40px 7px rgba(0,0,0,0.12)",
                "0px 10px 14px -6px rgba(0,0,0,0.2),0px 22px 35px 3px rgba(0,0,0,0.14),0px 8px 42px 7px rgba(0,0,0,0.12)",
                "0px 11px 14px -7px rgba(0,0,0,0.2),0px 23px 36px 3px rgba(0,0,0,0.14),0px 9px 44px 8px rgba(0,0,0,0.12)",
                "0px 11px 15px -7px rgba(0,0,0,0.2),0px 24px 38px 3px rgba(0,0,0,0.14),0px 9px 46px 8px rgba(0,0,0,0.12)",
              ],
              "shape": Object {
                "borderRadius": 4,
              },
              "spacing": Object {
                "unit": 8,
              },
              "transitions": Object {
                "create": [Function],
                "duration": Object {
                  "complex": 375,
                  "enteringScreen": 225,
                  "leavingScreen": 195,
                  "short": 250,
                  "shorter": 200,
                  "shortest": 150,
                  "standard": 300,
                },
                "easing": Object {
                  "easeIn": "cubic-bezier(0.4, 0, 1, 1)",
                  "easeInOut": "cubic-bezier(0.4, 0, 0.2, 1)",
                  "easeOut": "cubic-bezier(0.0, 0, 0.2, 1)",
                  "sharp": "cubic-bezier(0.4, 0, 0.6, 1)",
                },
                "getAutoHeightDuration": [Function],
              },
              "typography": Object {
                "body1": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.875rem",
                  "fontWeight": 400,
                  "lineHeight": "1.46429em",
                },
                "body1Next": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1rem",
                  "fontWeight": 400,
                  "letterSpacing": "0.00938em",
                  "lineHeight": 1.5,
                },
                "body2": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.875rem",
                  "fontWeight": 500,
                  "lineHeight": "1.71429em",
                },
                "body2Next": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.875rem",
                  "fontWeight": 400,
                  "letterSpacing": "0.01071em",
                  "lineHeight": 1.5,
                },
                "button": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.875rem",
                  "fontWeight": 500,
                  "textTransform": "uppercase",
                },
                "buttonNext": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.875rem",
                  "fontWeight": 500,
                  "letterSpacing": "0.02857em",
                  "lineHeight": 1.5,
                  "textTransform": "uppercase",
                },
                "caption": Object {
                  "color": "rgba(0, 0, 0, 0.54)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.75rem",
                  "fontWeight": 400,
                  "lineHeight": "1.375em",
                },
                "captionNext": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.75rem",
                  "fontWeight": 400,
                  "letterSpacing": "0.03333em",
                  "lineHeight": 1.66,
                },
                "display1": Object {
                  "color": "rgba(0, 0, 0, 0.54)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "2.125rem",
                  "fontWeight": 400,
                  "lineHeight": "1.20588em",
                },
                "display2": Object {
                  "color": "rgba(0, 0, 0, 0.54)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "2.8125rem",
                  "fontWeight": 400,
                  "lineHeight": "1.13333em",
                  "marginLeft": "-.02em",
                },
                "display3": Object {
                  "color": "rgba(0, 0, 0, 0.54)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "3.5rem",
                  "fontWeight": 400,
                  "letterSpacing": "-.02em",
                  "lineHeight": "1.30357em",
                  "marginLeft": "-.02em",
                },
                "display4": Object {
                  "color": "rgba(0, 0, 0, 0.54)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "7rem",
                  "fontWeight": 300,
                  "letterSpacing": "-.04em",
                  "lineHeight": "1.14286em",
                  "marginLeft": "-.04em",
                },
                "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                "fontSize": 14,
                "fontWeightLight": 300,
                "fontWeightMedium": 500,
                "fontWeightRegular": 400,
                "h1": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "6rem",
                  "fontWeight": 300,
                  "letterSpacing": "-0.01562em",
                  "lineHeight": 1,
                },
                "h2": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "3.75rem",
                  "fontWeight": 300,
                  "letterSpacing": "-0.00833em",
                  "lineHeight": 1,
                },
                "h3": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "3rem",
                  "fontWeight": 400,
                  "letterSpacing": "0em",
                  "lineHeight": 1.04,
                },
                "h4": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "2.125rem",
                  "fontWeight": 400,
                  "letterSpacing": "0.00735em",
                  "lineHeight": 1.17,
                },
                "h5": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1.5rem",
                  "fontWeight": 400,
                  "letterSpacing": "0em",
                  "lineHeight": 1.33,
                },
                "h6": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1.25rem",
                  "fontWeight": 500,
                  "letterSpacing": "0.0075em",
                  "lineHeight": 1.6,
                },
                "headline": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1.5rem",
                  "fontWeight": 400,
                  "lineHeight": "1.35417em",
                },
                "overline": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.75rem",
                  "fontWeight": 400,
                  "letterSpacing": "0.08333em",
                  "lineHeight": 2.66,
                  "textTransform": "uppercase",
                },
                "pxToRem": [Function],
                "round": [Function],
                "subheading": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1rem",
                  "fontWeight": 400,
                  "lineHeight": "1.5em",
                },
                "subtitle1": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1rem",
                  "fontWeight": 400,
                  "letterSpacing": "0.00938em",
                  "lineHeight": 1.75,
                },
                "subtitle2": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.875rem",
                  "fontWeight": 500,
                  "letterSpacing": "0.00714em",
                  "lineHeight": 1.57,
                },
                "title": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1.3125rem",
                  "fontWeight": 500,
                  "lineHeight": "1.16667em",
                },
                "useNextVariants": false,
              },
              "zIndex": Object {
                "appBar": 1100,
                "drawer": 1200,
                "mobileStepper": 1000,
                "modal": 1300,
                "snackbar": 1400,
                "tooltip": 1500,
              },
            }
          }
          title={
            <div>
              <div>
                Pending execution
              </div>
            </div>
          }
        >
          <span
            style={
              Object {
                "height": 18,
              }
            }
          >
            <pure(ScheduleIcon)
              style={
                Object {
                  "color": "#9aa0a6",
                  "height": 18,
                  "width": 18,
                }
              }
            />
          </span>
        </Tooltip>
      `);
    });

    it('handles RUNNING phase', () => {
      const tree = shallow(statusToIcon(NodePhase.RUNNING));
      expect(tree).toMatchInlineSnapshot(`
        <Tooltip
          TransitionComponent={[Function]}
          classes={
            Object {
              "popper": "MuiTooltip-popper-1",
              "popperInteractive": "MuiTooltip-popperInteractive-2",
              "tooltip": "MuiTooltip-tooltip-3",
              "tooltipPlacementBottom": "MuiTooltip-tooltipPlacementBottom-8",
              "tooltipPlacementLeft": "MuiTooltip-tooltipPlacementLeft-5",
              "tooltipPlacementRight": "MuiTooltip-tooltipPlacementRight-6",
              "tooltipPlacementTop": "MuiTooltip-tooltipPlacementTop-7",
              "touch": "MuiTooltip-touch-4",
            }
          }
          disableFocusListener={false}
          disableHoverListener={false}
          disableTouchListener={false}
          enterDelay={0}
          enterTouchDelay={1000}
          interactive={false}
          leaveDelay={0}
          leaveTouchDelay={1500}
          placement="bottom"
          theme={
            Object {
              "breakpoints": Object {
                "between": [Function],
                "down": [Function],
                "keys": Array [
                  "xs",
                  "sm",
                  "md",
                  "lg",
                  "xl",
                ],
                "only": [Function],
                "up": [Function],
                "values": Object {
                  "lg": 1280,
                  "md": 960,
                  "sm": 600,
                  "xl": 1920,
                  "xs": 0,
                },
                "width": [Function],
              },
              "direction": "ltr",
              "mixins": Object {
                "gutters": [Function],
                "toolbar": Object {
                  "@media (min-width:0px) and (orientation: landscape)": Object {
                    "minHeight": 48,
                  },
                  "@media (min-width:600px)": Object {
                    "minHeight": 64,
                  },
                  "minHeight": 56,
                },
              },
              "overrides": Object {},
              "palette": Object {
                "action": Object {
                  "active": "rgba(0, 0, 0, 0.54)",
                  "disabled": "rgba(0, 0, 0, 0.26)",
                  "disabledBackground": "rgba(0, 0, 0, 0.12)",
                  "hover": "rgba(0, 0, 0, 0.08)",
                  "hoverOpacity": 0.08,
                  "selected": "rgba(0, 0, 0, 0.14)",
                },
                "augmentColor": [Function],
                "background": Object {
                  "default": "#fafafa",
                  "paper": "#fff",
                },
                "common": Object {
                  "black": "#000",
                  "white": "#fff",
                },
                "contrastThreshold": 3,
                "divider": "rgba(0, 0, 0, 0.12)",
                "error": Object {
                  "contrastText": "#fff",
                  "dark": "#d32f2f",
                  "light": "#e57373",
                  "main": "#f44336",
                },
                "getContrastText": [Function],
                "grey": Object {
                  "100": "#f5f5f5",
                  "200": "#eeeeee",
                  "300": "#e0e0e0",
                  "400": "#bdbdbd",
                  "50": "#fafafa",
                  "500": "#9e9e9e",
                  "600": "#757575",
                  "700": "#616161",
                  "800": "#424242",
                  "900": "#212121",
                  "A100": "#d5d5d5",
                  "A200": "#aaaaaa",
                  "A400": "#303030",
                  "A700": "#616161",
                },
                "primary": Object {
                  "contrastText": "#fff",
                  "dark": "#303f9f",
                  "light": "#7986cb",
                  "main": "#3f51b5",
                },
                "secondary": Object {
                  "contrastText": "#fff",
                  "dark": "#c51162",
                  "light": "#ff4081",
                  "main": "#f50057",
                },
                "text": Object {
                  "disabled": "rgba(0, 0, 0, 0.38)",
                  "hint": "rgba(0, 0, 0, 0.38)",
                  "primary": "rgba(0, 0, 0, 0.87)",
                  "secondary": "rgba(0, 0, 0, 0.54)",
                },
                "tonalOffset": 0.2,
                "type": "light",
              },
              "props": Object {},
              "shadows": Array [
                "none",
                "0px 1px 3px 0px rgba(0,0,0,0.2),0px 1px 1px 0px rgba(0,0,0,0.14),0px 2px 1px -1px rgba(0,0,0,0.12)",
                "0px 1px 5px 0px rgba(0,0,0,0.2),0px 2px 2px 0px rgba(0,0,0,0.14),0px 3px 1px -2px rgba(0,0,0,0.12)",
                "0px 1px 8px 0px rgba(0,0,0,0.2),0px 3px 4px 0px rgba(0,0,0,0.14),0px 3px 3px -2px rgba(0,0,0,0.12)",
                "0px 2px 4px -1px rgba(0,0,0,0.2),0px 4px 5px 0px rgba(0,0,0,0.14),0px 1px 10px 0px rgba(0,0,0,0.12)",
                "0px 3px 5px -1px rgba(0,0,0,0.2),0px 5px 8px 0px rgba(0,0,0,0.14),0px 1px 14px 0px rgba(0,0,0,0.12)",
                "0px 3px 5px -1px rgba(0,0,0,0.2),0px 6px 10px 0px rgba(0,0,0,0.14),0px 1px 18px 0px rgba(0,0,0,0.12)",
                "0px 4px 5px -2px rgba(0,0,0,0.2),0px 7px 10px 1px rgba(0,0,0,0.14),0px 2px 16px 1px rgba(0,0,0,0.12)",
                "0px 5px 5px -3px rgba(0,0,0,0.2),0px 8px 10px 1px rgba(0,0,0,0.14),0px 3px 14px 2px rgba(0,0,0,0.12)",
                "0px 5px 6px -3px rgba(0,0,0,0.2),0px 9px 12px 1px rgba(0,0,0,0.14),0px 3px 16px 2px rgba(0,0,0,0.12)",
                "0px 6px 6px -3px rgba(0,0,0,0.2),0px 10px 14px 1px rgba(0,0,0,0.14),0px 4px 18px 3px rgba(0,0,0,0.12)",
                "0px 6px 7px -4px rgba(0,0,0,0.2),0px 11px 15px 1px rgba(0,0,0,0.14),0px 4px 20px 3px rgba(0,0,0,0.12)",
                "0px 7px 8px -4px rgba(0,0,0,0.2),0px 12px 17px 2px rgba(0,0,0,0.14),0px 5px 22px 4px rgba(0,0,0,0.12)",
                "0px 7px 8px -4px rgba(0,0,0,0.2),0px 13px 19px 2px rgba(0,0,0,0.14),0px 5px 24px 4px rgba(0,0,0,0.12)",
                "0px 7px 9px -4px rgba(0,0,0,0.2),0px 14px 21px 2px rgba(0,0,0,0.14),0px 5px 26px 4px rgba(0,0,0,0.12)",
                "0px 8px 9px -5px rgba(0,0,0,0.2),0px 15px 22px 2px rgba(0,0,0,0.14),0px 6px 28px 5px rgba(0,0,0,0.12)",
                "0px 8px 10px -5px rgba(0,0,0,0.2),0px 16px 24px 2px rgba(0,0,0,0.14),0px 6px 30px 5px rgba(0,0,0,0.12)",
                "0px 8px 11px -5px rgba(0,0,0,0.2),0px 17px 26px 2px rgba(0,0,0,0.14),0px 6px 32px 5px rgba(0,0,0,0.12)",
                "0px 9px 11px -5px rgba(0,0,0,0.2),0px 18px 28px 2px rgba(0,0,0,0.14),0px 7px 34px 6px rgba(0,0,0,0.12)",
                "0px 9px 12px -6px rgba(0,0,0,0.2),0px 19px 29px 2px rgba(0,0,0,0.14),0px 7px 36px 6px rgba(0,0,0,0.12)",
                "0px 10px 13px -6px rgba(0,0,0,0.2),0px 20px 31px 3px rgba(0,0,0,0.14),0px 8px 38px 7px rgba(0,0,0,0.12)",
                "0px 10px 13px -6px rgba(0,0,0,0.2),0px 21px 33px 3px rgba(0,0,0,0.14),0px 8px 40px 7px rgba(0,0,0,0.12)",
                "0px 10px 14px -6px rgba(0,0,0,0.2),0px 22px 35px 3px rgba(0,0,0,0.14),0px 8px 42px 7px rgba(0,0,0,0.12)",
                "0px 11px 14px -7px rgba(0,0,0,0.2),0px 23px 36px 3px rgba(0,0,0,0.14),0px 9px 44px 8px rgba(0,0,0,0.12)",
                "0px 11px 15px -7px rgba(0,0,0,0.2),0px 24px 38px 3px rgba(0,0,0,0.14),0px 9px 46px 8px rgba(0,0,0,0.12)",
              ],
              "shape": Object {
                "borderRadius": 4,
              },
              "spacing": Object {
                "unit": 8,
              },
              "transitions": Object {
                "create": [Function],
                "duration": Object {
                  "complex": 375,
                  "enteringScreen": 225,
                  "leavingScreen": 195,
                  "short": 250,
                  "shorter": 200,
                  "shortest": 150,
                  "standard": 300,
                },
                "easing": Object {
                  "easeIn": "cubic-bezier(0.4, 0, 1, 1)",
                  "easeInOut": "cubic-bezier(0.4, 0, 0.2, 1)",
                  "easeOut": "cubic-bezier(0.0, 0, 0.2, 1)",
                  "sharp": "cubic-bezier(0.4, 0, 0.6, 1)",
                },
                "getAutoHeightDuration": [Function],
              },
              "typography": Object {
                "body1": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.875rem",
                  "fontWeight": 400,
                  "lineHeight": "1.46429em",
                },
                "body1Next": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1rem",
                  "fontWeight": 400,
                  "letterSpacing": "0.00938em",
                  "lineHeight": 1.5,
                },
                "body2": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.875rem",
                  "fontWeight": 500,
                  "lineHeight": "1.71429em",
                },
                "body2Next": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.875rem",
                  "fontWeight": 400,
                  "letterSpacing": "0.01071em",
                  "lineHeight": 1.5,
                },
                "button": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.875rem",
                  "fontWeight": 500,
                  "textTransform": "uppercase",
                },
                "buttonNext": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.875rem",
                  "fontWeight": 500,
                  "letterSpacing": "0.02857em",
                  "lineHeight": 1.5,
                  "textTransform": "uppercase",
                },
                "caption": Object {
                  "color": "rgba(0, 0, 0, 0.54)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.75rem",
                  "fontWeight": 400,
                  "lineHeight": "1.375em",
                },
                "captionNext": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.75rem",
                  "fontWeight": 400,
                  "letterSpacing": "0.03333em",
                  "lineHeight": 1.66,
                },
                "display1": Object {
                  "color": "rgba(0, 0, 0, 0.54)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "2.125rem",
                  "fontWeight": 400,
                  "lineHeight": "1.20588em",
                },
                "display2": Object {
                  "color": "rgba(0, 0, 0, 0.54)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "2.8125rem",
                  "fontWeight": 400,
                  "lineHeight": "1.13333em",
                  "marginLeft": "-.02em",
                },
                "display3": Object {
                  "color": "rgba(0, 0, 0, 0.54)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "3.5rem",
                  "fontWeight": 400,
                  "letterSpacing": "-.02em",
                  "lineHeight": "1.30357em",
                  "marginLeft": "-.02em",
                },
                "display4": Object {
                  "color": "rgba(0, 0, 0, 0.54)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "7rem",
                  "fontWeight": 300,
                  "letterSpacing": "-.04em",
                  "lineHeight": "1.14286em",
                  "marginLeft": "-.04em",
                },
                "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                "fontSize": 14,
                "fontWeightLight": 300,
                "fontWeightMedium": 500,
                "fontWeightRegular": 400,
                "h1": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "6rem",
                  "fontWeight": 300,
                  "letterSpacing": "-0.01562em",
                  "lineHeight": 1,
                },
                "h2": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "3.75rem",
                  "fontWeight": 300,
                  "letterSpacing": "-0.00833em",
                  "lineHeight": 1,
                },
                "h3": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "3rem",
                  "fontWeight": 400,
                  "letterSpacing": "0em",
                  "lineHeight": 1.04,
                },
                "h4": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "2.125rem",
                  "fontWeight": 400,
                  "letterSpacing": "0.00735em",
                  "lineHeight": 1.17,
                },
                "h5": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1.5rem",
                  "fontWeight": 400,
                  "letterSpacing": "0em",
                  "lineHeight": 1.33,
                },
                "h6": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1.25rem",
                  "fontWeight": 500,
                  "letterSpacing": "0.0075em",
                  "lineHeight": 1.6,
                },
                "headline": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1.5rem",
                  "fontWeight": 400,
                  "lineHeight": "1.35417em",
                },
                "overline": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.75rem",
                  "fontWeight": 400,
                  "letterSpacing": "0.08333em",
                  "lineHeight": 2.66,
                  "textTransform": "uppercase",
                },
                "pxToRem": [Function],
                "round": [Function],
                "subheading": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1rem",
                  "fontWeight": 400,
                  "lineHeight": "1.5em",
                },
                "subtitle1": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1rem",
                  "fontWeight": 400,
                  "letterSpacing": "0.00938em",
                  "lineHeight": 1.75,
                },
                "subtitle2": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.875rem",
                  "fontWeight": 500,
                  "letterSpacing": "0.00714em",
                  "lineHeight": 1.57,
                },
                "title": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1.3125rem",
                  "fontWeight": 500,
                  "lineHeight": "1.16667em",
                },
                "useNextVariants": false,
              },
              "zIndex": Object {
                "appBar": 1100,
                "drawer": 1200,
                "mobileStepper": 1000,
                "modal": 1300,
                "snackbar": 1400,
                "tooltip": 1500,
              },
            }
          }
          title={
            <div>
              <div>
                Running
              </div>
            </div>
          }
        >
          <span
            style={
              Object {
                "height": 18,
              }
            }
          >
            <StatusRunning
              style={
                Object {
                  "color": "#4285f4",
                  "height": 18,
                  "width": 18,
                }
              }
            />
          </span>
        </Tooltip>
      `);
    });

    it('handles TERMINATING phase', () => {
      const tree = shallow(statusToIcon(NodePhase.TERMINATING));
      expect(tree).toMatchInlineSnapshot(`
        <Tooltip
          TransitionComponent={[Function]}
          classes={
            Object {
              "popper": "MuiTooltip-popper-1",
              "popperInteractive": "MuiTooltip-popperInteractive-2",
              "tooltip": "MuiTooltip-tooltip-3",
              "tooltipPlacementBottom": "MuiTooltip-tooltipPlacementBottom-8",
              "tooltipPlacementLeft": "MuiTooltip-tooltipPlacementLeft-5",
              "tooltipPlacementRight": "MuiTooltip-tooltipPlacementRight-6",
              "tooltipPlacementTop": "MuiTooltip-tooltipPlacementTop-7",
              "touch": "MuiTooltip-touch-4",
            }
          }
          disableFocusListener={false}
          disableHoverListener={false}
          disableTouchListener={false}
          enterDelay={0}
          enterTouchDelay={1000}
          interactive={false}
          leaveDelay={0}
          leaveTouchDelay={1500}
          placement="bottom"
          theme={
            Object {
              "breakpoints": Object {
                "between": [Function],
                "down": [Function],
                "keys": Array [
                  "xs",
                  "sm",
                  "md",
                  "lg",
                  "xl",
                ],
                "only": [Function],
                "up": [Function],
                "values": Object {
                  "lg": 1280,
                  "md": 960,
                  "sm": 600,
                  "xl": 1920,
                  "xs": 0,
                },
                "width": [Function],
              },
              "direction": "ltr",
              "mixins": Object {
                "gutters": [Function],
                "toolbar": Object {
                  "@media (min-width:0px) and (orientation: landscape)": Object {
                    "minHeight": 48,
                  },
                  "@media (min-width:600px)": Object {
                    "minHeight": 64,
                  },
                  "minHeight": 56,
                },
              },
              "overrides": Object {},
              "palette": Object {
                "action": Object {
                  "active": "rgba(0, 0, 0, 0.54)",
                  "disabled": "rgba(0, 0, 0, 0.26)",
                  "disabledBackground": "rgba(0, 0, 0, 0.12)",
                  "hover": "rgba(0, 0, 0, 0.08)",
                  "hoverOpacity": 0.08,
                  "selected": "rgba(0, 0, 0, 0.14)",
                },
                "augmentColor": [Function],
                "background": Object {
                  "default": "#fafafa",
                  "paper": "#fff",
                },
                "common": Object {
                  "black": "#000",
                  "white": "#fff",
                },
                "contrastThreshold": 3,
                "divider": "rgba(0, 0, 0, 0.12)",
                "error": Object {
                  "contrastText": "#fff",
                  "dark": "#d32f2f",
                  "light": "#e57373",
                  "main": "#f44336",
                },
                "getContrastText": [Function],
                "grey": Object {
                  "100": "#f5f5f5",
                  "200": "#eeeeee",
                  "300": "#e0e0e0",
                  "400": "#bdbdbd",
                  "50": "#fafafa",
                  "500": "#9e9e9e",
                  "600": "#757575",
                  "700": "#616161",
                  "800": "#424242",
                  "900": "#212121",
                  "A100": "#d5d5d5",
                  "A200": "#aaaaaa",
                  "A400": "#303030",
                  "A700": "#616161",
                },
                "primary": Object {
                  "contrastText": "#fff",
                  "dark": "#303f9f",
                  "light": "#7986cb",
                  "main": "#3f51b5",
                },
                "secondary": Object {
                  "contrastText": "#fff",
                  "dark": "#c51162",
                  "light": "#ff4081",
                  "main": "#f50057",
                },
                "text": Object {
                  "disabled": "rgba(0, 0, 0, 0.38)",
                  "hint": "rgba(0, 0, 0, 0.38)",
                  "primary": "rgba(0, 0, 0, 0.87)",
                  "secondary": "rgba(0, 0, 0, 0.54)",
                },
                "tonalOffset": 0.2,
                "type": "light",
              },
              "props": Object {},
              "shadows": Array [
                "none",
                "0px 1px 3px 0px rgba(0,0,0,0.2),0px 1px 1px 0px rgba(0,0,0,0.14),0px 2px 1px -1px rgba(0,0,0,0.12)",
                "0px 1px 5px 0px rgba(0,0,0,0.2),0px 2px 2px 0px rgba(0,0,0,0.14),0px 3px 1px -2px rgba(0,0,0,0.12)",
                "0px 1px 8px 0px rgba(0,0,0,0.2),0px 3px 4px 0px rgba(0,0,0,0.14),0px 3px 3px -2px rgba(0,0,0,0.12)",
                "0px 2px 4px -1px rgba(0,0,0,0.2),0px 4px 5px 0px rgba(0,0,0,0.14),0px 1px 10px 0px rgba(0,0,0,0.12)",
                "0px 3px 5px -1px rgba(0,0,0,0.2),0px 5px 8px 0px rgba(0,0,0,0.14),0px 1px 14px 0px rgba(0,0,0,0.12)",
                "0px 3px 5px -1px rgba(0,0,0,0.2),0px 6px 10px 0px rgba(0,0,0,0.14),0px 1px 18px 0px rgba(0,0,0,0.12)",
                "0px 4px 5px -2px rgba(0,0,0,0.2),0px 7px 10px 1px rgba(0,0,0,0.14),0px 2px 16px 1px rgba(0,0,0,0.12)",
                "0px 5px 5px -3px rgba(0,0,0,0.2),0px 8px 10px 1px rgba(0,0,0,0.14),0px 3px 14px 2px rgba(0,0,0,0.12)",
                "0px 5px 6px -3px rgba(0,0,0,0.2),0px 9px 12px 1px rgba(0,0,0,0.14),0px 3px 16px 2px rgba(0,0,0,0.12)",
                "0px 6px 6px -3px rgba(0,0,0,0.2),0px 10px 14px 1px rgba(0,0,0,0.14),0px 4px 18px 3px rgba(0,0,0,0.12)",
                "0px 6px 7px -4px rgba(0,0,0,0.2),0px 11px 15px 1px rgba(0,0,0,0.14),0px 4px 20px 3px rgba(0,0,0,0.12)",
                "0px 7px 8px -4px rgba(0,0,0,0.2),0px 12px 17px 2px rgba(0,0,0,0.14),0px 5px 22px 4px rgba(0,0,0,0.12)",
                "0px 7px 8px -4px rgba(0,0,0,0.2),0px 13px 19px 2px rgba(0,0,0,0.14),0px 5px 24px 4px rgba(0,0,0,0.12)",
                "0px 7px 9px -4px rgba(0,0,0,0.2),0px 14px 21px 2px rgba(0,0,0,0.14),0px 5px 26px 4px rgba(0,0,0,0.12)",
                "0px 8px 9px -5px rgba(0,0,0,0.2),0px 15px 22px 2px rgba(0,0,0,0.14),0px 6px 28px 5px rgba(0,0,0,0.12)",
                "0px 8px 10px -5px rgba(0,0,0,0.2),0px 16px 24px 2px rgba(0,0,0,0.14),0px 6px 30px 5px rgba(0,0,0,0.12)",
                "0px 8px 11px -5px rgba(0,0,0,0.2),0px 17px 26px 2px rgba(0,0,0,0.14),0px 6px 32px 5px rgba(0,0,0,0.12)",
                "0px 9px 11px -5px rgba(0,0,0,0.2),0px 18px 28px 2px rgba(0,0,0,0.14),0px 7px 34px 6px rgba(0,0,0,0.12)",
                "0px 9px 12px -6px rgba(0,0,0,0.2),0px 19px 29px 2px rgba(0,0,0,0.14),0px 7px 36px 6px rgba(0,0,0,0.12)",
                "0px 10px 13px -6px rgba(0,0,0,0.2),0px 20px 31px 3px rgba(0,0,0,0.14),0px 8px 38px 7px rgba(0,0,0,0.12)",
                "0px 10px 13px -6px rgba(0,0,0,0.2),0px 21px 33px 3px rgba(0,0,0,0.14),0px 8px 40px 7px rgba(0,0,0,0.12)",
                "0px 10px 14px -6px rgba(0,0,0,0.2),0px 22px 35px 3px rgba(0,0,0,0.14),0px 8px 42px 7px rgba(0,0,0,0.12)",
                "0px 11px 14px -7px rgba(0,0,0,0.2),0px 23px 36px 3px rgba(0,0,0,0.14),0px 9px 44px 8px rgba(0,0,0,0.12)",
                "0px 11px 15px -7px rgba(0,0,0,0.2),0px 24px 38px 3px rgba(0,0,0,0.14),0px 9px 46px 8px rgba(0,0,0,0.12)",
              ],
              "shape": Object {
                "borderRadius": 4,
              },
              "spacing": Object {
                "unit": 8,
              },
              "transitions": Object {
                "create": [Function],
                "duration": Object {
                  "complex": 375,
                  "enteringScreen": 225,
                  "leavingScreen": 195,
                  "short": 250,
                  "shorter": 200,
                  "shortest": 150,
                  "standard": 300,
                },
                "easing": Object {
                  "easeIn": "cubic-bezier(0.4, 0, 1, 1)",
                  "easeInOut": "cubic-bezier(0.4, 0, 0.2, 1)",
                  "easeOut": "cubic-bezier(0.0, 0, 0.2, 1)",
                  "sharp": "cubic-bezier(0.4, 0, 0.6, 1)",
                },
                "getAutoHeightDuration": [Function],
              },
              "typography": Object {
                "body1": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.875rem",
                  "fontWeight": 400,
                  "lineHeight": "1.46429em",
                },
                "body1Next": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1rem",
                  "fontWeight": 400,
                  "letterSpacing": "0.00938em",
                  "lineHeight": 1.5,
                },
                "body2": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.875rem",
                  "fontWeight": 500,
                  "lineHeight": "1.71429em",
                },
                "body2Next": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.875rem",
                  "fontWeight": 400,
                  "letterSpacing": "0.01071em",
                  "lineHeight": 1.5,
                },
                "button": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.875rem",
                  "fontWeight": 500,
                  "textTransform": "uppercase",
                },
                "buttonNext": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.875rem",
                  "fontWeight": 500,
                  "letterSpacing": "0.02857em",
                  "lineHeight": 1.5,
                  "textTransform": "uppercase",
                },
                "caption": Object {
                  "color": "rgba(0, 0, 0, 0.54)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.75rem",
                  "fontWeight": 400,
                  "lineHeight": "1.375em",
                },
                "captionNext": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.75rem",
                  "fontWeight": 400,
                  "letterSpacing": "0.03333em",
                  "lineHeight": 1.66,
                },
                "display1": Object {
                  "color": "rgba(0, 0, 0, 0.54)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "2.125rem",
                  "fontWeight": 400,
                  "lineHeight": "1.20588em",
                },
                "display2": Object {
                  "color": "rgba(0, 0, 0, 0.54)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "2.8125rem",
                  "fontWeight": 400,
                  "lineHeight": "1.13333em",
                  "marginLeft": "-.02em",
                },
                "display3": Object {
                  "color": "rgba(0, 0, 0, 0.54)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "3.5rem",
                  "fontWeight": 400,
                  "letterSpacing": "-.02em",
                  "lineHeight": "1.30357em",
                  "marginLeft": "-.02em",
                },
                "display4": Object {
                  "color": "rgba(0, 0, 0, 0.54)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "7rem",
                  "fontWeight": 300,
                  "letterSpacing": "-.04em",
                  "lineHeight": "1.14286em",
                  "marginLeft": "-.04em",
                },
                "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                "fontSize": 14,
                "fontWeightLight": 300,
                "fontWeightMedium": 500,
                "fontWeightRegular": 400,
                "h1": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "6rem",
                  "fontWeight": 300,
                  "letterSpacing": "-0.01562em",
                  "lineHeight": 1,
                },
                "h2": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "3.75rem",
                  "fontWeight": 300,
                  "letterSpacing": "-0.00833em",
                  "lineHeight": 1,
                },
                "h3": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "3rem",
                  "fontWeight": 400,
                  "letterSpacing": "0em",
                  "lineHeight": 1.04,
                },
                "h4": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "2.125rem",
                  "fontWeight": 400,
                  "letterSpacing": "0.00735em",
                  "lineHeight": 1.17,
                },
                "h5": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1.5rem",
                  "fontWeight": 400,
                  "letterSpacing": "0em",
                  "lineHeight": 1.33,
                },
                "h6": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1.25rem",
                  "fontWeight": 500,
                  "letterSpacing": "0.0075em",
                  "lineHeight": 1.6,
                },
                "headline": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1.5rem",
                  "fontWeight": 400,
                  "lineHeight": "1.35417em",
                },
                "overline": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.75rem",
                  "fontWeight": 400,
                  "letterSpacing": "0.08333em",
                  "lineHeight": 2.66,
                  "textTransform": "uppercase",
                },
                "pxToRem": [Function],
                "round": [Function],
                "subheading": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1rem",
                  "fontWeight": 400,
                  "lineHeight": "1.5em",
                },
                "subtitle1": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1rem",
                  "fontWeight": 400,
                  "letterSpacing": "0.00938em",
                  "lineHeight": 1.75,
                },
                "subtitle2": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.875rem",
                  "fontWeight": 500,
                  "letterSpacing": "0.00714em",
                  "lineHeight": 1.57,
                },
                "title": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1.3125rem",
                  "fontWeight": 500,
                  "lineHeight": "1.16667em",
                },
                "useNextVariants": false,
              },
              "zIndex": Object {
                "appBar": 1100,
                "drawer": 1200,
                "mobileStepper": 1000,
                "modal": 1300,
                "snackbar": 1400,
                "tooltip": 1500,
              },
            }
          }
          title={
            <div>
              <div>
                Run is terminating
              </div>
            </div>
          }
        >
          <span
            style={
              Object {
                "height": 18,
              }
            }
          >
            <StatusRunning
              style={
                Object {
                  "color": "#4285f4",
                  "height": 18,
                  "width": 18,
                }
              }
            />
          </span>
        </Tooltip>
      `);
    });

    it('handles SKIPPED phase', () => {
      const tree = shallow(statusToIcon(NodePhase.SKIPPED));
      expect(tree).toMatchInlineSnapshot(`
        <Tooltip
          TransitionComponent={[Function]}
          classes={
            Object {
              "popper": "MuiTooltip-popper-1",
              "popperInteractive": "MuiTooltip-popperInteractive-2",
              "tooltip": "MuiTooltip-tooltip-3",
              "tooltipPlacementBottom": "MuiTooltip-tooltipPlacementBottom-8",
              "tooltipPlacementLeft": "MuiTooltip-tooltipPlacementLeft-5",
              "tooltipPlacementRight": "MuiTooltip-tooltipPlacementRight-6",
              "tooltipPlacementTop": "MuiTooltip-tooltipPlacementTop-7",
              "touch": "MuiTooltip-touch-4",
            }
          }
          disableFocusListener={false}
          disableHoverListener={false}
          disableTouchListener={false}
          enterDelay={0}
          enterTouchDelay={1000}
          interactive={false}
          leaveDelay={0}
          leaveTouchDelay={1500}
          placement="bottom"
          theme={
            Object {
              "breakpoints": Object {
                "between": [Function],
                "down": [Function],
                "keys": Array [
                  "xs",
                  "sm",
                  "md",
                  "lg",
                  "xl",
                ],
                "only": [Function],
                "up": [Function],
                "values": Object {
                  "lg": 1280,
                  "md": 960,
                  "sm": 600,
                  "xl": 1920,
                  "xs": 0,
                },
                "width": [Function],
              },
              "direction": "ltr",
              "mixins": Object {
                "gutters": [Function],
                "toolbar": Object {
                  "@media (min-width:0px) and (orientation: landscape)": Object {
                    "minHeight": 48,
                  },
                  "@media (min-width:600px)": Object {
                    "minHeight": 64,
                  },
                  "minHeight": 56,
                },
              },
              "overrides": Object {},
              "palette": Object {
                "action": Object {
                  "active": "rgba(0, 0, 0, 0.54)",
                  "disabled": "rgba(0, 0, 0, 0.26)",
                  "disabledBackground": "rgba(0, 0, 0, 0.12)",
                  "hover": "rgba(0, 0, 0, 0.08)",
                  "hoverOpacity": 0.08,
                  "selected": "rgba(0, 0, 0, 0.14)",
                },
                "augmentColor": [Function],
                "background": Object {
                  "default": "#fafafa",
                  "paper": "#fff",
                },
                "common": Object {
                  "black": "#000",
                  "white": "#fff",
                },
                "contrastThreshold": 3,
                "divider": "rgba(0, 0, 0, 0.12)",
                "error": Object {
                  "contrastText": "#fff",
                  "dark": "#d32f2f",
                  "light": "#e57373",
                  "main": "#f44336",
                },
                "getContrastText": [Function],
                "grey": Object {
                  "100": "#f5f5f5",
                  "200": "#eeeeee",
                  "300": "#e0e0e0",
                  "400": "#bdbdbd",
                  "50": "#fafafa",
                  "500": "#9e9e9e",
                  "600": "#757575",
                  "700": "#616161",
                  "800": "#424242",
                  "900": "#212121",
                  "A100": "#d5d5d5",
                  "A200": "#aaaaaa",
                  "A400": "#303030",
                  "A700": "#616161",
                },
                "primary": Object {
                  "contrastText": "#fff",
                  "dark": "#303f9f",
                  "light": "#7986cb",
                  "main": "#3f51b5",
                },
                "secondary": Object {
                  "contrastText": "#fff",
                  "dark": "#c51162",
                  "light": "#ff4081",
                  "main": "#f50057",
                },
                "text": Object {
                  "disabled": "rgba(0, 0, 0, 0.38)",
                  "hint": "rgba(0, 0, 0, 0.38)",
                  "primary": "rgba(0, 0, 0, 0.87)",
                  "secondary": "rgba(0, 0, 0, 0.54)",
                },
                "tonalOffset": 0.2,
                "type": "light",
              },
              "props": Object {},
              "shadows": Array [
                "none",
                "0px 1px 3px 0px rgba(0,0,0,0.2),0px 1px 1px 0px rgba(0,0,0,0.14),0px 2px 1px -1px rgba(0,0,0,0.12)",
                "0px 1px 5px 0px rgba(0,0,0,0.2),0px 2px 2px 0px rgba(0,0,0,0.14),0px 3px 1px -2px rgba(0,0,0,0.12)",
                "0px 1px 8px 0px rgba(0,0,0,0.2),0px 3px 4px 0px rgba(0,0,0,0.14),0px 3px 3px -2px rgba(0,0,0,0.12)",
                "0px 2px 4px -1px rgba(0,0,0,0.2),0px 4px 5px 0px rgba(0,0,0,0.14),0px 1px 10px 0px rgba(0,0,0,0.12)",
                "0px 3px 5px -1px rgba(0,0,0,0.2),0px 5px 8px 0px rgba(0,0,0,0.14),0px 1px 14px 0px rgba(0,0,0,0.12)",
                "0px 3px 5px -1px rgba(0,0,0,0.2),0px 6px 10px 0px rgba(0,0,0,0.14),0px 1px 18px 0px rgba(0,0,0,0.12)",
                "0px 4px 5px -2px rgba(0,0,0,0.2),0px 7px 10px 1px rgba(0,0,0,0.14),0px 2px 16px 1px rgba(0,0,0,0.12)",
                "0px 5px 5px -3px rgba(0,0,0,0.2),0px 8px 10px 1px rgba(0,0,0,0.14),0px 3px 14px 2px rgba(0,0,0,0.12)",
                "0px 5px 6px -3px rgba(0,0,0,0.2),0px 9px 12px 1px rgba(0,0,0,0.14),0px 3px 16px 2px rgba(0,0,0,0.12)",
                "0px 6px 6px -3px rgba(0,0,0,0.2),0px 10px 14px 1px rgba(0,0,0,0.14),0px 4px 18px 3px rgba(0,0,0,0.12)",
                "0px 6px 7px -4px rgba(0,0,0,0.2),0px 11px 15px 1px rgba(0,0,0,0.14),0px 4px 20px 3px rgba(0,0,0,0.12)",
                "0px 7px 8px -4px rgba(0,0,0,0.2),0px 12px 17px 2px rgba(0,0,0,0.14),0px 5px 22px 4px rgba(0,0,0,0.12)",
                "0px 7px 8px -4px rgba(0,0,0,0.2),0px 13px 19px 2px rgba(0,0,0,0.14),0px 5px 24px 4px rgba(0,0,0,0.12)",
                "0px 7px 9px -4px rgba(0,0,0,0.2),0px 14px 21px 2px rgba(0,0,0,0.14),0px 5px 26px 4px rgba(0,0,0,0.12)",
                "0px 8px 9px -5px rgba(0,0,0,0.2),0px 15px 22px 2px rgba(0,0,0,0.14),0px 6px 28px 5px rgba(0,0,0,0.12)",
                "0px 8px 10px -5px rgba(0,0,0,0.2),0px 16px 24px 2px rgba(0,0,0,0.14),0px 6px 30px 5px rgba(0,0,0,0.12)",
                "0px 8px 11px -5px rgba(0,0,0,0.2),0px 17px 26px 2px rgba(0,0,0,0.14),0px 6px 32px 5px rgba(0,0,0,0.12)",
                "0px 9px 11px -5px rgba(0,0,0,0.2),0px 18px 28px 2px rgba(0,0,0,0.14),0px 7px 34px 6px rgba(0,0,0,0.12)",
                "0px 9px 12px -6px rgba(0,0,0,0.2),0px 19px 29px 2px rgba(0,0,0,0.14),0px 7px 36px 6px rgba(0,0,0,0.12)",
                "0px 10px 13px -6px rgba(0,0,0,0.2),0px 20px 31px 3px rgba(0,0,0,0.14),0px 8px 38px 7px rgba(0,0,0,0.12)",
                "0px 10px 13px -6px rgba(0,0,0,0.2),0px 21px 33px 3px rgba(0,0,0,0.14),0px 8px 40px 7px rgba(0,0,0,0.12)",
                "0px 10px 14px -6px rgba(0,0,0,0.2),0px 22px 35px 3px rgba(0,0,0,0.14),0px 8px 42px 7px rgba(0,0,0,0.12)",
                "0px 11px 14px -7px rgba(0,0,0,0.2),0px 23px 36px 3px rgba(0,0,0,0.14),0px 9px 44px 8px rgba(0,0,0,0.12)",
                "0px 11px 15px -7px rgba(0,0,0,0.2),0px 24px 38px 3px rgba(0,0,0,0.14),0px 9px 46px 8px rgba(0,0,0,0.12)",
              ],
              "shape": Object {
                "borderRadius": 4,
              },
              "spacing": Object {
                "unit": 8,
              },
              "transitions": Object {
                "create": [Function],
                "duration": Object {
                  "complex": 375,
                  "enteringScreen": 225,
                  "leavingScreen": 195,
                  "short": 250,
                  "shorter": 200,
                  "shortest": 150,
                  "standard": 300,
                },
                "easing": Object {
                  "easeIn": "cubic-bezier(0.4, 0, 1, 1)",
                  "easeInOut": "cubic-bezier(0.4, 0, 0.2, 1)",
                  "easeOut": "cubic-bezier(0.0, 0, 0.2, 1)",
                  "sharp": "cubic-bezier(0.4, 0, 0.6, 1)",
                },
                "getAutoHeightDuration": [Function],
              },
              "typography": Object {
                "body1": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.875rem",
                  "fontWeight": 400,
                  "lineHeight": "1.46429em",
                },
                "body1Next": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1rem",
                  "fontWeight": 400,
                  "letterSpacing": "0.00938em",
                  "lineHeight": 1.5,
                },
                "body2": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.875rem",
                  "fontWeight": 500,
                  "lineHeight": "1.71429em",
                },
                "body2Next": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.875rem",
                  "fontWeight": 400,
                  "letterSpacing": "0.01071em",
                  "lineHeight": 1.5,
                },
                "button": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.875rem",
                  "fontWeight": 500,
                  "textTransform": "uppercase",
                },
                "buttonNext": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.875rem",
                  "fontWeight": 500,
                  "letterSpacing": "0.02857em",
                  "lineHeight": 1.5,
                  "textTransform": "uppercase",
                },
                "caption": Object {
                  "color": "rgba(0, 0, 0, 0.54)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.75rem",
                  "fontWeight": 400,
                  "lineHeight": "1.375em",
                },
                "captionNext": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.75rem",
                  "fontWeight": 400,
                  "letterSpacing": "0.03333em",
                  "lineHeight": 1.66,
                },
                "display1": Object {
                  "color": "rgba(0, 0, 0, 0.54)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "2.125rem",
                  "fontWeight": 400,
                  "lineHeight": "1.20588em",
                },
                "display2": Object {
                  "color": "rgba(0, 0, 0, 0.54)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "2.8125rem",
                  "fontWeight": 400,
                  "lineHeight": "1.13333em",
                  "marginLeft": "-.02em",
                },
                "display3": Object {
                  "color": "rgba(0, 0, 0, 0.54)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "3.5rem",
                  "fontWeight": 400,
                  "letterSpacing": "-.02em",
                  "lineHeight": "1.30357em",
                  "marginLeft": "-.02em",
                },
                "display4": Object {
                  "color": "rgba(0, 0, 0, 0.54)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "7rem",
                  "fontWeight": 300,
                  "letterSpacing": "-.04em",
                  "lineHeight": "1.14286em",
                  "marginLeft": "-.04em",
                },
                "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                "fontSize": 14,
                "fontWeightLight": 300,
                "fontWeightMedium": 500,
                "fontWeightRegular": 400,
                "h1": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "6rem",
                  "fontWeight": 300,
                  "letterSpacing": "-0.01562em",
                  "lineHeight": 1,
                },
                "h2": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "3.75rem",
                  "fontWeight": 300,
                  "letterSpacing": "-0.00833em",
                  "lineHeight": 1,
                },
                "h3": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "3rem",
                  "fontWeight": 400,
                  "letterSpacing": "0em",
                  "lineHeight": 1.04,
                },
                "h4": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "2.125rem",
                  "fontWeight": 400,
                  "letterSpacing": "0.00735em",
                  "lineHeight": 1.17,
                },
                "h5": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1.5rem",
                  "fontWeight": 400,
                  "letterSpacing": "0em",
                  "lineHeight": 1.33,
                },
                "h6": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1.25rem",
                  "fontWeight": 500,
                  "letterSpacing": "0.0075em",
                  "lineHeight": 1.6,
                },
                "headline": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1.5rem",
                  "fontWeight": 400,
                  "lineHeight": "1.35417em",
                },
                "overline": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.75rem",
                  "fontWeight": 400,
                  "letterSpacing": "0.08333em",
                  "lineHeight": 2.66,
                  "textTransform": "uppercase",
                },
                "pxToRem": [Function],
                "round": [Function],
                "subheading": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1rem",
                  "fontWeight": 400,
                  "lineHeight": "1.5em",
                },
                "subtitle1": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1rem",
                  "fontWeight": 400,
                  "letterSpacing": "0.00938em",
                  "lineHeight": 1.75,
                },
                "subtitle2": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.875rem",
                  "fontWeight": 500,
                  "letterSpacing": "0.00714em",
                  "lineHeight": 1.57,
                },
                "title": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1.3125rem",
                  "fontWeight": 500,
                  "lineHeight": "1.16667em",
                },
                "useNextVariants": false,
              },
              "zIndex": Object {
                "appBar": 1100,
                "drawer": 1200,
                "mobileStepper": 1000,
                "modal": 1300,
                "snackbar": 1400,
                "tooltip": 1500,
              },
            }
          }
          title={
            <div>
              <div>
                Execution has been skipped for this resource
              </div>
            </div>
          }
        >
          <span
            style={
              Object {
                "height": 18,
              }
            }
          >
            <pure(SkipNextIcon)
              style={
                Object {
                  "color": "#5f6368",
                  "height": 18,
                  "width": 18,
                }
              }
            />
          </span>
        </Tooltip>
      `);
    });

    it('handles SUCCEEDED phase', () => {
      const tree = shallow(statusToIcon(NodePhase.SUCCEEDED));
      expect(tree).toMatchInlineSnapshot(`
        <Tooltip
          TransitionComponent={[Function]}
          classes={
            Object {
              "popper": "MuiTooltip-popper-1",
              "popperInteractive": "MuiTooltip-popperInteractive-2",
              "tooltip": "MuiTooltip-tooltip-3",
              "tooltipPlacementBottom": "MuiTooltip-tooltipPlacementBottom-8",
              "tooltipPlacementLeft": "MuiTooltip-tooltipPlacementLeft-5",
              "tooltipPlacementRight": "MuiTooltip-tooltipPlacementRight-6",
              "tooltipPlacementTop": "MuiTooltip-tooltipPlacementTop-7",
              "touch": "MuiTooltip-touch-4",
            }
          }
          disableFocusListener={false}
          disableHoverListener={false}
          disableTouchListener={false}
          enterDelay={0}
          enterTouchDelay={1000}
          interactive={false}
          leaveDelay={0}
          leaveTouchDelay={1500}
          placement="bottom"
          theme={
            Object {
              "breakpoints": Object {
                "between": [Function],
                "down": [Function],
                "keys": Array [
                  "xs",
                  "sm",
                  "md",
                  "lg",
                  "xl",
                ],
                "only": [Function],
                "up": [Function],
                "values": Object {
                  "lg": 1280,
                  "md": 960,
                  "sm": 600,
                  "xl": 1920,
                  "xs": 0,
                },
                "width": [Function],
              },
              "direction": "ltr",
              "mixins": Object {
                "gutters": [Function],
                "toolbar": Object {
                  "@media (min-width:0px) and (orientation: landscape)": Object {
                    "minHeight": 48,
                  },
                  "@media (min-width:600px)": Object {
                    "minHeight": 64,
                  },
                  "minHeight": 56,
                },
              },
              "overrides": Object {},
              "palette": Object {
                "action": Object {
                  "active": "rgba(0, 0, 0, 0.54)",
                  "disabled": "rgba(0, 0, 0, 0.26)",
                  "disabledBackground": "rgba(0, 0, 0, 0.12)",
                  "hover": "rgba(0, 0, 0, 0.08)",
                  "hoverOpacity": 0.08,
                  "selected": "rgba(0, 0, 0, 0.14)",
                },
                "augmentColor": [Function],
                "background": Object {
                  "default": "#fafafa",
                  "paper": "#fff",
                },
                "common": Object {
                  "black": "#000",
                  "white": "#fff",
                },
                "contrastThreshold": 3,
                "divider": "rgba(0, 0, 0, 0.12)",
                "error": Object {
                  "contrastText": "#fff",
                  "dark": "#d32f2f",
                  "light": "#e57373",
                  "main": "#f44336",
                },
                "getContrastText": [Function],
                "grey": Object {
                  "100": "#f5f5f5",
                  "200": "#eeeeee",
                  "300": "#e0e0e0",
                  "400": "#bdbdbd",
                  "50": "#fafafa",
                  "500": "#9e9e9e",
                  "600": "#757575",
                  "700": "#616161",
                  "800": "#424242",
                  "900": "#212121",
                  "A100": "#d5d5d5",
                  "A200": "#aaaaaa",
                  "A400": "#303030",
                  "A700": "#616161",
                },
                "primary": Object {
                  "contrastText": "#fff",
                  "dark": "#303f9f",
                  "light": "#7986cb",
                  "main": "#3f51b5",
                },
                "secondary": Object {
                  "contrastText": "#fff",
                  "dark": "#c51162",
                  "light": "#ff4081",
                  "main": "#f50057",
                },
                "text": Object {
                  "disabled": "rgba(0, 0, 0, 0.38)",
                  "hint": "rgba(0, 0, 0, 0.38)",
                  "primary": "rgba(0, 0, 0, 0.87)",
                  "secondary": "rgba(0, 0, 0, 0.54)",
                },
                "tonalOffset": 0.2,
                "type": "light",
              },
              "props": Object {},
              "shadows": Array [
                "none",
                "0px 1px 3px 0px rgba(0,0,0,0.2),0px 1px 1px 0px rgba(0,0,0,0.14),0px 2px 1px -1px rgba(0,0,0,0.12)",
                "0px 1px 5px 0px rgba(0,0,0,0.2),0px 2px 2px 0px rgba(0,0,0,0.14),0px 3px 1px -2px rgba(0,0,0,0.12)",
                "0px 1px 8px 0px rgba(0,0,0,0.2),0px 3px 4px 0px rgba(0,0,0,0.14),0px 3px 3px -2px rgba(0,0,0,0.12)",
                "0px 2px 4px -1px rgba(0,0,0,0.2),0px 4px 5px 0px rgba(0,0,0,0.14),0px 1px 10px 0px rgba(0,0,0,0.12)",
                "0px 3px 5px -1px rgba(0,0,0,0.2),0px 5px 8px 0px rgba(0,0,0,0.14),0px 1px 14px 0px rgba(0,0,0,0.12)",
                "0px 3px 5px -1px rgba(0,0,0,0.2),0px 6px 10px 0px rgba(0,0,0,0.14),0px 1px 18px 0px rgba(0,0,0,0.12)",
                "0px 4px 5px -2px rgba(0,0,0,0.2),0px 7px 10px 1px rgba(0,0,0,0.14),0px 2px 16px 1px rgba(0,0,0,0.12)",
                "0px 5px 5px -3px rgba(0,0,0,0.2),0px 8px 10px 1px rgba(0,0,0,0.14),0px 3px 14px 2px rgba(0,0,0,0.12)",
                "0px 5px 6px -3px rgba(0,0,0,0.2),0px 9px 12px 1px rgba(0,0,0,0.14),0px 3px 16px 2px rgba(0,0,0,0.12)",
                "0px 6px 6px -3px rgba(0,0,0,0.2),0px 10px 14px 1px rgba(0,0,0,0.14),0px 4px 18px 3px rgba(0,0,0,0.12)",
                "0px 6px 7px -4px rgba(0,0,0,0.2),0px 11px 15px 1px rgba(0,0,0,0.14),0px 4px 20px 3px rgba(0,0,0,0.12)",
                "0px 7px 8px -4px rgba(0,0,0,0.2),0px 12px 17px 2px rgba(0,0,0,0.14),0px 5px 22px 4px rgba(0,0,0,0.12)",
                "0px 7px 8px -4px rgba(0,0,0,0.2),0px 13px 19px 2px rgba(0,0,0,0.14),0px 5px 24px 4px rgba(0,0,0,0.12)",
                "0px 7px 9px -4px rgba(0,0,0,0.2),0px 14px 21px 2px rgba(0,0,0,0.14),0px 5px 26px 4px rgba(0,0,0,0.12)",
                "0px 8px 9px -5px rgba(0,0,0,0.2),0px 15px 22px 2px rgba(0,0,0,0.14),0px 6px 28px 5px rgba(0,0,0,0.12)",
                "0px 8px 10px -5px rgba(0,0,0,0.2),0px 16px 24px 2px rgba(0,0,0,0.14),0px 6px 30px 5px rgba(0,0,0,0.12)",
                "0px 8px 11px -5px rgba(0,0,0,0.2),0px 17px 26px 2px rgba(0,0,0,0.14),0px 6px 32px 5px rgba(0,0,0,0.12)",
                "0px 9px 11px -5px rgba(0,0,0,0.2),0px 18px 28px 2px rgba(0,0,0,0.14),0px 7px 34px 6px rgba(0,0,0,0.12)",
                "0px 9px 12px -6px rgba(0,0,0,0.2),0px 19px 29px 2px rgba(0,0,0,0.14),0px 7px 36px 6px rgba(0,0,0,0.12)",
                "0px 10px 13px -6px rgba(0,0,0,0.2),0px 20px 31px 3px rgba(0,0,0,0.14),0px 8px 38px 7px rgba(0,0,0,0.12)",
                "0px 10px 13px -6px rgba(0,0,0,0.2),0px 21px 33px 3px rgba(0,0,0,0.14),0px 8px 40px 7px rgba(0,0,0,0.12)",
                "0px 10px 14px -6px rgba(0,0,0,0.2),0px 22px 35px 3px rgba(0,0,0,0.14),0px 8px 42px 7px rgba(0,0,0,0.12)",
                "0px 11px 14px -7px rgba(0,0,0,0.2),0px 23px 36px 3px rgba(0,0,0,0.14),0px 9px 44px 8px rgba(0,0,0,0.12)",
                "0px 11px 15px -7px rgba(0,0,0,0.2),0px 24px 38px 3px rgba(0,0,0,0.14),0px 9px 46px 8px rgba(0,0,0,0.12)",
              ],
              "shape": Object {
                "borderRadius": 4,
              },
              "spacing": Object {
                "unit": 8,
              },
              "transitions": Object {
                "create": [Function],
                "duration": Object {
                  "complex": 375,
                  "enteringScreen": 225,
                  "leavingScreen": 195,
                  "short": 250,
                  "shorter": 200,
                  "shortest": 150,
                  "standard": 300,
                },
                "easing": Object {
                  "easeIn": "cubic-bezier(0.4, 0, 1, 1)",
                  "easeInOut": "cubic-bezier(0.4, 0, 0.2, 1)",
                  "easeOut": "cubic-bezier(0.0, 0, 0.2, 1)",
                  "sharp": "cubic-bezier(0.4, 0, 0.6, 1)",
                },
                "getAutoHeightDuration": [Function],
              },
              "typography": Object {
                "body1": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.875rem",
                  "fontWeight": 400,
                  "lineHeight": "1.46429em",
                },
                "body1Next": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1rem",
                  "fontWeight": 400,
                  "letterSpacing": "0.00938em",
                  "lineHeight": 1.5,
                },
                "body2": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.875rem",
                  "fontWeight": 500,
                  "lineHeight": "1.71429em",
                },
                "body2Next": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.875rem",
                  "fontWeight": 400,
                  "letterSpacing": "0.01071em",
                  "lineHeight": 1.5,
                },
                "button": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.875rem",
                  "fontWeight": 500,
                  "textTransform": "uppercase",
                },
                "buttonNext": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.875rem",
                  "fontWeight": 500,
                  "letterSpacing": "0.02857em",
                  "lineHeight": 1.5,
                  "textTransform": "uppercase",
                },
                "caption": Object {
                  "color": "rgba(0, 0, 0, 0.54)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.75rem",
                  "fontWeight": 400,
                  "lineHeight": "1.375em",
                },
                "captionNext": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.75rem",
                  "fontWeight": 400,
                  "letterSpacing": "0.03333em",
                  "lineHeight": 1.66,
                },
                "display1": Object {
                  "color": "rgba(0, 0, 0, 0.54)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "2.125rem",
                  "fontWeight": 400,
                  "lineHeight": "1.20588em",
                },
                "display2": Object {
                  "color": "rgba(0, 0, 0, 0.54)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "2.8125rem",
                  "fontWeight": 400,
                  "lineHeight": "1.13333em",
                  "marginLeft": "-.02em",
                },
                "display3": Object {
                  "color": "rgba(0, 0, 0, 0.54)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "3.5rem",
                  "fontWeight": 400,
                  "letterSpacing": "-.02em",
                  "lineHeight": "1.30357em",
                  "marginLeft": "-.02em",
                },
                "display4": Object {
                  "color": "rgba(0, 0, 0, 0.54)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "7rem",
                  "fontWeight": 300,
                  "letterSpacing": "-.04em",
                  "lineHeight": "1.14286em",
                  "marginLeft": "-.04em",
                },
                "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                "fontSize": 14,
                "fontWeightLight": 300,
                "fontWeightMedium": 500,
                "fontWeightRegular": 400,
                "h1": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "6rem",
                  "fontWeight": 300,
                  "letterSpacing": "-0.01562em",
                  "lineHeight": 1,
                },
                "h2": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "3.75rem",
                  "fontWeight": 300,
                  "letterSpacing": "-0.00833em",
                  "lineHeight": 1,
                },
                "h3": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "3rem",
                  "fontWeight": 400,
                  "letterSpacing": "0em",
                  "lineHeight": 1.04,
                },
                "h4": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "2.125rem",
                  "fontWeight": 400,
                  "letterSpacing": "0.00735em",
                  "lineHeight": 1.17,
                },
                "h5": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1.5rem",
                  "fontWeight": 400,
                  "letterSpacing": "0em",
                  "lineHeight": 1.33,
                },
                "h6": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1.25rem",
                  "fontWeight": 500,
                  "letterSpacing": "0.0075em",
                  "lineHeight": 1.6,
                },
                "headline": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1.5rem",
                  "fontWeight": 400,
                  "lineHeight": "1.35417em",
                },
                "overline": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.75rem",
                  "fontWeight": 400,
                  "letterSpacing": "0.08333em",
                  "lineHeight": 2.66,
                  "textTransform": "uppercase",
                },
                "pxToRem": [Function],
                "round": [Function],
                "subheading": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1rem",
                  "fontWeight": 400,
                  "lineHeight": "1.5em",
                },
                "subtitle1": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1rem",
                  "fontWeight": 400,
                  "letterSpacing": "0.00938em",
                  "lineHeight": 1.75,
                },
                "subtitle2": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.875rem",
                  "fontWeight": 500,
                  "letterSpacing": "0.00714em",
                  "lineHeight": 1.57,
                },
                "title": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1.3125rem",
                  "fontWeight": 500,
                  "lineHeight": "1.16667em",
                },
                "useNextVariants": false,
              },
              "zIndex": Object {
                "appBar": 1100,
                "drawer": 1200,
                "mobileStepper": 1000,
                "modal": 1300,
                "snackbar": 1400,
                "tooltip": 1500,
              },
            }
          }
          title={
            <div>
              <div>
                Executed successfully
              </div>
            </div>
          }
        >
          <span
            style={
              Object {
                "height": 18,
              }
            }
          >
            <pure(CheckCircleIcon)
              style={
                Object {
                  "color": "#34a853",
                  "height": 18,
                  "width": 18,
                }
              }
            />
          </span>
        </Tooltip>
      `);
    });

    it('handles CACHED phase', () => {
      const tree = shallow(statusToIcon(NodePhase.CACHED));
      expect(tree).toMatchInlineSnapshot(`
        <Tooltip
          TransitionComponent={[Function]}
          classes={
            Object {
              "popper": "MuiTooltip-popper-1",
              "popperInteractive": "MuiTooltip-popperInteractive-2",
              "tooltip": "MuiTooltip-tooltip-3",
              "tooltipPlacementBottom": "MuiTooltip-tooltipPlacementBottom-8",
              "tooltipPlacementLeft": "MuiTooltip-tooltipPlacementLeft-5",
              "tooltipPlacementRight": "MuiTooltip-tooltipPlacementRight-6",
              "tooltipPlacementTop": "MuiTooltip-tooltipPlacementTop-7",
              "touch": "MuiTooltip-touch-4",
            }
          }
          disableFocusListener={false}
          disableHoverListener={false}
          disableTouchListener={false}
          enterDelay={0}
          enterTouchDelay={1000}
          interactive={false}
          leaveDelay={0}
          leaveTouchDelay={1500}
          placement="bottom"
          theme={
            Object {
              "breakpoints": Object {
                "between": [Function],
                "down": [Function],
                "keys": Array [
                  "xs",
                  "sm",
                  "md",
                  "lg",
                  "xl",
                ],
                "only": [Function],
                "up": [Function],
                "values": Object {
                  "lg": 1280,
                  "md": 960,
                  "sm": 600,
                  "xl": 1920,
                  "xs": 0,
                },
                "width": [Function],
              },
              "direction": "ltr",
              "mixins": Object {
                "gutters": [Function],
                "toolbar": Object {
                  "@media (min-width:0px) and (orientation: landscape)": Object {
                    "minHeight": 48,
                  },
                  "@media (min-width:600px)": Object {
                    "minHeight": 64,
                  },
                  "minHeight": 56,
                },
              },
              "overrides": Object {},
              "palette": Object {
                "action": Object {
                  "active": "rgba(0, 0, 0, 0.54)",
                  "disabled": "rgba(0, 0, 0, 0.26)",
                  "disabledBackground": "rgba(0, 0, 0, 0.12)",
                  "hover": "rgba(0, 0, 0, 0.08)",
                  "hoverOpacity": 0.08,
                  "selected": "rgba(0, 0, 0, 0.14)",
                },
                "augmentColor": [Function],
                "background": Object {
                  "default": "#fafafa",
                  "paper": "#fff",
                },
                "common": Object {
                  "black": "#000",
                  "white": "#fff",
                },
                "contrastThreshold": 3,
                "divider": "rgba(0, 0, 0, 0.12)",
                "error": Object {
                  "contrastText": "#fff",
                  "dark": "#d32f2f",
                  "light": "#e57373",
                  "main": "#f44336",
                },
                "getContrastText": [Function],
                "grey": Object {
                  "100": "#f5f5f5",
                  "200": "#eeeeee",
                  "300": "#e0e0e0",
                  "400": "#bdbdbd",
                  "50": "#fafafa",
                  "500": "#9e9e9e",
                  "600": "#757575",
                  "700": "#616161",
                  "800": "#424242",
                  "900": "#212121",
                  "A100": "#d5d5d5",
                  "A200": "#aaaaaa",
                  "A400": "#303030",
                  "A700": "#616161",
                },
                "primary": Object {
                  "contrastText": "#fff",
                  "dark": "#303f9f",
                  "light": "#7986cb",
                  "main": "#3f51b5",
                },
                "secondary": Object {
                  "contrastText": "#fff",
                  "dark": "#c51162",
                  "light": "#ff4081",
                  "main": "#f50057",
                },
                "text": Object {
                  "disabled": "rgba(0, 0, 0, 0.38)",
                  "hint": "rgba(0, 0, 0, 0.38)",
                  "primary": "rgba(0, 0, 0, 0.87)",
                  "secondary": "rgba(0, 0, 0, 0.54)",
                },
                "tonalOffset": 0.2,
                "type": "light",
              },
              "props": Object {},
              "shadows": Array [
                "none",
                "0px 1px 3px 0px rgba(0,0,0,0.2),0px 1px 1px 0px rgba(0,0,0,0.14),0px 2px 1px -1px rgba(0,0,0,0.12)",
                "0px 1px 5px 0px rgba(0,0,0,0.2),0px 2px 2px 0px rgba(0,0,0,0.14),0px 3px 1px -2px rgba(0,0,0,0.12)",
                "0px 1px 8px 0px rgba(0,0,0,0.2),0px 3px 4px 0px rgba(0,0,0,0.14),0px 3px 3px -2px rgba(0,0,0,0.12)",
                "0px 2px 4px -1px rgba(0,0,0,0.2),0px 4px 5px 0px rgba(0,0,0,0.14),0px 1px 10px 0px rgba(0,0,0,0.12)",
                "0px 3px 5px -1px rgba(0,0,0,0.2),0px 5px 8px 0px rgba(0,0,0,0.14),0px 1px 14px 0px rgba(0,0,0,0.12)",
                "0px 3px 5px -1px rgba(0,0,0,0.2),0px 6px 10px 0px rgba(0,0,0,0.14),0px 1px 18px 0px rgba(0,0,0,0.12)",
                "0px 4px 5px -2px rgba(0,0,0,0.2),0px 7px 10px 1px rgba(0,0,0,0.14),0px 2px 16px 1px rgba(0,0,0,0.12)",
                "0px 5px 5px -3px rgba(0,0,0,0.2),0px 8px 10px 1px rgba(0,0,0,0.14),0px 3px 14px 2px rgba(0,0,0,0.12)",
                "0px 5px 6px -3px rgba(0,0,0,0.2),0px 9px 12px 1px rgba(0,0,0,0.14),0px 3px 16px 2px rgba(0,0,0,0.12)",
                "0px 6px 6px -3px rgba(0,0,0,0.2),0px 10px 14px 1px rgba(0,0,0,0.14),0px 4px 18px 3px rgba(0,0,0,0.12)",
                "0px 6px 7px -4px rgba(0,0,0,0.2),0px 11px 15px 1px rgba(0,0,0,0.14),0px 4px 20px 3px rgba(0,0,0,0.12)",
                "0px 7px 8px -4px rgba(0,0,0,0.2),0px 12px 17px 2px rgba(0,0,0,0.14),0px 5px 22px 4px rgba(0,0,0,0.12)",
                "0px 7px 8px -4px rgba(0,0,0,0.2),0px 13px 19px 2px rgba(0,0,0,0.14),0px 5px 24px 4px rgba(0,0,0,0.12)",
                "0px 7px 9px -4px rgba(0,0,0,0.2),0px 14px 21px 2px rgba(0,0,0,0.14),0px 5px 26px 4px rgba(0,0,0,0.12)",
                "0px 8px 9px -5px rgba(0,0,0,0.2),0px 15px 22px 2px rgba(0,0,0,0.14),0px 6px 28px 5px rgba(0,0,0,0.12)",
                "0px 8px 10px -5px rgba(0,0,0,0.2),0px 16px 24px 2px rgba(0,0,0,0.14),0px 6px 30px 5px rgba(0,0,0,0.12)",
                "0px 8px 11px -5px rgba(0,0,0,0.2),0px 17px 26px 2px rgba(0,0,0,0.14),0px 6px 32px 5px rgba(0,0,0,0.12)",
                "0px 9px 11px -5px rgba(0,0,0,0.2),0px 18px 28px 2px rgba(0,0,0,0.14),0px 7px 34px 6px rgba(0,0,0,0.12)",
                "0px 9px 12px -6px rgba(0,0,0,0.2),0px 19px 29px 2px rgba(0,0,0,0.14),0px 7px 36px 6px rgba(0,0,0,0.12)",
                "0px 10px 13px -6px rgba(0,0,0,0.2),0px 20px 31px 3px rgba(0,0,0,0.14),0px 8px 38px 7px rgba(0,0,0,0.12)",
                "0px 10px 13px -6px rgba(0,0,0,0.2),0px 21px 33px 3px rgba(0,0,0,0.14),0px 8px 40px 7px rgba(0,0,0,0.12)",
                "0px 10px 14px -6px rgba(0,0,0,0.2),0px 22px 35px 3px rgba(0,0,0,0.14),0px 8px 42px 7px rgba(0,0,0,0.12)",
                "0px 11px 14px -7px rgba(0,0,0,0.2),0px 23px 36px 3px rgba(0,0,0,0.14),0px 9px 44px 8px rgba(0,0,0,0.12)",
                "0px 11px 15px -7px rgba(0,0,0,0.2),0px 24px 38px 3px rgba(0,0,0,0.14),0px 9px 46px 8px rgba(0,0,0,0.12)",
              ],
              "shape": Object {
                "borderRadius": 4,
              },
              "spacing": Object {
                "unit": 8,
              },
              "transitions": Object {
                "create": [Function],
                "duration": Object {
                  "complex": 375,
                  "enteringScreen": 225,
                  "leavingScreen": 195,
                  "short": 250,
                  "shorter": 200,
                  "shortest": 150,
                  "standard": 300,
                },
                "easing": Object {
                  "easeIn": "cubic-bezier(0.4, 0, 1, 1)",
                  "easeInOut": "cubic-bezier(0.4, 0, 0.2, 1)",
                  "easeOut": "cubic-bezier(0.0, 0, 0.2, 1)",
                  "sharp": "cubic-bezier(0.4, 0, 0.6, 1)",
                },
                "getAutoHeightDuration": [Function],
              },
              "typography": Object {
                "body1": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.875rem",
                  "fontWeight": 400,
                  "lineHeight": "1.46429em",
                },
                "body1Next": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1rem",
                  "fontWeight": 400,
                  "letterSpacing": "0.00938em",
                  "lineHeight": 1.5,
                },
                "body2": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.875rem",
                  "fontWeight": 500,
                  "lineHeight": "1.71429em",
                },
                "body2Next": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.875rem",
                  "fontWeight": 400,
                  "letterSpacing": "0.01071em",
                  "lineHeight": 1.5,
                },
                "button": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.875rem",
                  "fontWeight": 500,
                  "textTransform": "uppercase",
                },
                "buttonNext": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.875rem",
                  "fontWeight": 500,
                  "letterSpacing": "0.02857em",
                  "lineHeight": 1.5,
                  "textTransform": "uppercase",
                },
                "caption": Object {
                  "color": "rgba(0, 0, 0, 0.54)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.75rem",
                  "fontWeight": 400,
                  "lineHeight": "1.375em",
                },
                "captionNext": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.75rem",
                  "fontWeight": 400,
                  "letterSpacing": "0.03333em",
                  "lineHeight": 1.66,
                },
                "display1": Object {
                  "color": "rgba(0, 0, 0, 0.54)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "2.125rem",
                  "fontWeight": 400,
                  "lineHeight": "1.20588em",
                },
                "display2": Object {
                  "color": "rgba(0, 0, 0, 0.54)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "2.8125rem",
                  "fontWeight": 400,
                  "lineHeight": "1.13333em",
                  "marginLeft": "-.02em",
                },
                "display3": Object {
                  "color": "rgba(0, 0, 0, 0.54)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "3.5rem",
                  "fontWeight": 400,
                  "letterSpacing": "-.02em",
                  "lineHeight": "1.30357em",
                  "marginLeft": "-.02em",
                },
                "display4": Object {
                  "color": "rgba(0, 0, 0, 0.54)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "7rem",
                  "fontWeight": 300,
                  "letterSpacing": "-.04em",
                  "lineHeight": "1.14286em",
                  "marginLeft": "-.04em",
                },
                "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                "fontSize": 14,
                "fontWeightLight": 300,
                "fontWeightMedium": 500,
                "fontWeightRegular": 400,
                "h1": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "6rem",
                  "fontWeight": 300,
                  "letterSpacing": "-0.01562em",
                  "lineHeight": 1,
                },
                "h2": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "3.75rem",
                  "fontWeight": 300,
                  "letterSpacing": "-0.00833em",
                  "lineHeight": 1,
                },
                "h3": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "3rem",
                  "fontWeight": 400,
                  "letterSpacing": "0em",
                  "lineHeight": 1.04,
                },
                "h4": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "2.125rem",
                  "fontWeight": 400,
                  "letterSpacing": "0.00735em",
                  "lineHeight": 1.17,
                },
                "h5": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1.5rem",
                  "fontWeight": 400,
                  "letterSpacing": "0em",
                  "lineHeight": 1.33,
                },
                "h6": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1.25rem",
                  "fontWeight": 500,
                  "letterSpacing": "0.0075em",
                  "lineHeight": 1.6,
                },
                "headline": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1.5rem",
                  "fontWeight": 400,
                  "lineHeight": "1.35417em",
                },
                "overline": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.75rem",
                  "fontWeight": 400,
                  "letterSpacing": "0.08333em",
                  "lineHeight": 2.66,
                  "textTransform": "uppercase",
                },
                "pxToRem": [Function],
                "round": [Function],
                "subheading": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1rem",
                  "fontWeight": 400,
                  "lineHeight": "1.5em",
                },
                "subtitle1": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1rem",
                  "fontWeight": 400,
                  "letterSpacing": "0.00938em",
                  "lineHeight": 1.75,
                },
                "subtitle2": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.875rem",
                  "fontWeight": 500,
                  "letterSpacing": "0.00714em",
                  "lineHeight": 1.57,
                },
                "title": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1.3125rem",
                  "fontWeight": 500,
                  "lineHeight": "1.16667em",
                },
                "useNextVariants": false,
              },
              "zIndex": Object {
                "appBar": 1100,
                "drawer": 1200,
                "mobileStepper": 1000,
                "modal": 1300,
                "snackbar": 1400,
                "tooltip": 1500,
              },
            }
          }
          title={
            <div>
              <div>
                Execution was skipped and outputs were taken from cache
              </div>
            </div>
          }
        >
          <span
            style={
              Object {
                "height": 18,
              }
            }
          >
            <StatusCached
              style={
                Object {
                  "color": "#34a853",
                  "height": 18,
                  "width": 18,
                }
              }
            />
          </span>
        </Tooltip>
      `);
    });

    it('handles TERMINATED phase', () => {
      const tree = shallow(statusToIcon(NodePhase.TERMINATED));
      expect(tree).toMatchInlineSnapshot(`
        <Tooltip
          TransitionComponent={[Function]}
          classes={
            Object {
              "popper": "MuiTooltip-popper-1",
              "popperInteractive": "MuiTooltip-popperInteractive-2",
              "tooltip": "MuiTooltip-tooltip-3",
              "tooltipPlacementBottom": "MuiTooltip-tooltipPlacementBottom-8",
              "tooltipPlacementLeft": "MuiTooltip-tooltipPlacementLeft-5",
              "tooltipPlacementRight": "MuiTooltip-tooltipPlacementRight-6",
              "tooltipPlacementTop": "MuiTooltip-tooltipPlacementTop-7",
              "touch": "MuiTooltip-touch-4",
            }
          }
          disableFocusListener={false}
          disableHoverListener={false}
          disableTouchListener={false}
          enterDelay={0}
          enterTouchDelay={1000}
          interactive={false}
          leaveDelay={0}
          leaveTouchDelay={1500}
          placement="bottom"
          theme={
            Object {
              "breakpoints": Object {
                "between": [Function],
                "down": [Function],
                "keys": Array [
                  "xs",
                  "sm",
                  "md",
                  "lg",
                  "xl",
                ],
                "only": [Function],
                "up": [Function],
                "values": Object {
                  "lg": 1280,
                  "md": 960,
                  "sm": 600,
                  "xl": 1920,
                  "xs": 0,
                },
                "width": [Function],
              },
              "direction": "ltr",
              "mixins": Object {
                "gutters": [Function],
                "toolbar": Object {
                  "@media (min-width:0px) and (orientation: landscape)": Object {
                    "minHeight": 48,
                  },
                  "@media (min-width:600px)": Object {
                    "minHeight": 64,
                  },
                  "minHeight": 56,
                },
              },
              "overrides": Object {},
              "palette": Object {
                "action": Object {
                  "active": "rgba(0, 0, 0, 0.54)",
                  "disabled": "rgba(0, 0, 0, 0.26)",
                  "disabledBackground": "rgba(0, 0, 0, 0.12)",
                  "hover": "rgba(0, 0, 0, 0.08)",
                  "hoverOpacity": 0.08,
                  "selected": "rgba(0, 0, 0, 0.14)",
                },
                "augmentColor": [Function],
                "background": Object {
                  "default": "#fafafa",
                  "paper": "#fff",
                },
                "common": Object {
                  "black": "#000",
                  "white": "#fff",
                },
                "contrastThreshold": 3,
                "divider": "rgba(0, 0, 0, 0.12)",
                "error": Object {
                  "contrastText": "#fff",
                  "dark": "#d32f2f",
                  "light": "#e57373",
                  "main": "#f44336",
                },
                "getContrastText": [Function],
                "grey": Object {
                  "100": "#f5f5f5",
                  "200": "#eeeeee",
                  "300": "#e0e0e0",
                  "400": "#bdbdbd",
                  "50": "#fafafa",
                  "500": "#9e9e9e",
                  "600": "#757575",
                  "700": "#616161",
                  "800": "#424242",
                  "900": "#212121",
                  "A100": "#d5d5d5",
                  "A200": "#aaaaaa",
                  "A400": "#303030",
                  "A700": "#616161",
                },
                "primary": Object {
                  "contrastText": "#fff",
                  "dark": "#303f9f",
                  "light": "#7986cb",
                  "main": "#3f51b5",
                },
                "secondary": Object {
                  "contrastText": "#fff",
                  "dark": "#c51162",
                  "light": "#ff4081",
                  "main": "#f50057",
                },
                "text": Object {
                  "disabled": "rgba(0, 0, 0, 0.38)",
                  "hint": "rgba(0, 0, 0, 0.38)",
                  "primary": "rgba(0, 0, 0, 0.87)",
                  "secondary": "rgba(0, 0, 0, 0.54)",
                },
                "tonalOffset": 0.2,
                "type": "light",
              },
              "props": Object {},
              "shadows": Array [
                "none",
                "0px 1px 3px 0px rgba(0,0,0,0.2),0px 1px 1px 0px rgba(0,0,0,0.14),0px 2px 1px -1px rgba(0,0,0,0.12)",
                "0px 1px 5px 0px rgba(0,0,0,0.2),0px 2px 2px 0px rgba(0,0,0,0.14),0px 3px 1px -2px rgba(0,0,0,0.12)",
                "0px 1px 8px 0px rgba(0,0,0,0.2),0px 3px 4px 0px rgba(0,0,0,0.14),0px 3px 3px -2px rgba(0,0,0,0.12)",
                "0px 2px 4px -1px rgba(0,0,0,0.2),0px 4px 5px 0px rgba(0,0,0,0.14),0px 1px 10px 0px rgba(0,0,0,0.12)",
                "0px 3px 5px -1px rgba(0,0,0,0.2),0px 5px 8px 0px rgba(0,0,0,0.14),0px 1px 14px 0px rgba(0,0,0,0.12)",
                "0px 3px 5px -1px rgba(0,0,0,0.2),0px 6px 10px 0px rgba(0,0,0,0.14),0px 1px 18px 0px rgba(0,0,0,0.12)",
                "0px 4px 5px -2px rgba(0,0,0,0.2),0px 7px 10px 1px rgba(0,0,0,0.14),0px 2px 16px 1px rgba(0,0,0,0.12)",
                "0px 5px 5px -3px rgba(0,0,0,0.2),0px 8px 10px 1px rgba(0,0,0,0.14),0px 3px 14px 2px rgba(0,0,0,0.12)",
                "0px 5px 6px -3px rgba(0,0,0,0.2),0px 9px 12px 1px rgba(0,0,0,0.14),0px 3px 16px 2px rgba(0,0,0,0.12)",
                "0px 6px 6px -3px rgba(0,0,0,0.2),0px 10px 14px 1px rgba(0,0,0,0.14),0px 4px 18px 3px rgba(0,0,0,0.12)",
                "0px 6px 7px -4px rgba(0,0,0,0.2),0px 11px 15px 1px rgba(0,0,0,0.14),0px 4px 20px 3px rgba(0,0,0,0.12)",
                "0px 7px 8px -4px rgba(0,0,0,0.2),0px 12px 17px 2px rgba(0,0,0,0.14),0px 5px 22px 4px rgba(0,0,0,0.12)",
                "0px 7px 8px -4px rgba(0,0,0,0.2),0px 13px 19px 2px rgba(0,0,0,0.14),0px 5px 24px 4px rgba(0,0,0,0.12)",
                "0px 7px 9px -4px rgba(0,0,0,0.2),0px 14px 21px 2px rgba(0,0,0,0.14),0px 5px 26px 4px rgba(0,0,0,0.12)",
                "0px 8px 9px -5px rgba(0,0,0,0.2),0px 15px 22px 2px rgba(0,0,0,0.14),0px 6px 28px 5px rgba(0,0,0,0.12)",
                "0px 8px 10px -5px rgba(0,0,0,0.2),0px 16px 24px 2px rgba(0,0,0,0.14),0px 6px 30px 5px rgba(0,0,0,0.12)",
                "0px 8px 11px -5px rgba(0,0,0,0.2),0px 17px 26px 2px rgba(0,0,0,0.14),0px 6px 32px 5px rgba(0,0,0,0.12)",
                "0px 9px 11px -5px rgba(0,0,0,0.2),0px 18px 28px 2px rgba(0,0,0,0.14),0px 7px 34px 6px rgba(0,0,0,0.12)",
                "0px 9px 12px -6px rgba(0,0,0,0.2),0px 19px 29px 2px rgba(0,0,0,0.14),0px 7px 36px 6px rgba(0,0,0,0.12)",
                "0px 10px 13px -6px rgba(0,0,0,0.2),0px 20px 31px 3px rgba(0,0,0,0.14),0px 8px 38px 7px rgba(0,0,0,0.12)",
                "0px 10px 13px -6px rgba(0,0,0,0.2),0px 21px 33px 3px rgba(0,0,0,0.14),0px 8px 40px 7px rgba(0,0,0,0.12)",
                "0px 10px 14px -6px rgba(0,0,0,0.2),0px 22px 35px 3px rgba(0,0,0,0.14),0px 8px 42px 7px rgba(0,0,0,0.12)",
                "0px 11px 14px -7px rgba(0,0,0,0.2),0px 23px 36px 3px rgba(0,0,0,0.14),0px 9px 44px 8px rgba(0,0,0,0.12)",
                "0px 11px 15px -7px rgba(0,0,0,0.2),0px 24px 38px 3px rgba(0,0,0,0.14),0px 9px 46px 8px rgba(0,0,0,0.12)",
              ],
              "shape": Object {
                "borderRadius": 4,
              },
              "spacing": Object {
                "unit": 8,
              },
              "transitions": Object {
                "create": [Function],
                "duration": Object {
                  "complex": 375,
                  "enteringScreen": 225,
                  "leavingScreen": 195,
                  "short": 250,
                  "shorter": 200,
                  "shortest": 150,
                  "standard": 300,
                },
                "easing": Object {
                  "easeIn": "cubic-bezier(0.4, 0, 1, 1)",
                  "easeInOut": "cubic-bezier(0.4, 0, 0.2, 1)",
                  "easeOut": "cubic-bezier(0.0, 0, 0.2, 1)",
                  "sharp": "cubic-bezier(0.4, 0, 0.6, 1)",
                },
                "getAutoHeightDuration": [Function],
              },
              "typography": Object {
                "body1": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.875rem",
                  "fontWeight": 400,
                  "lineHeight": "1.46429em",
                },
                "body1Next": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1rem",
                  "fontWeight": 400,
                  "letterSpacing": "0.00938em",
                  "lineHeight": 1.5,
                },
                "body2": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.875rem",
                  "fontWeight": 500,
                  "lineHeight": "1.71429em",
                },
                "body2Next": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.875rem",
                  "fontWeight": 400,
                  "letterSpacing": "0.01071em",
                  "lineHeight": 1.5,
                },
                "button": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.875rem",
                  "fontWeight": 500,
                  "textTransform": "uppercase",
                },
                "buttonNext": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.875rem",
                  "fontWeight": 500,
                  "letterSpacing": "0.02857em",
                  "lineHeight": 1.5,
                  "textTransform": "uppercase",
                },
                "caption": Object {
                  "color": "rgba(0, 0, 0, 0.54)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.75rem",
                  "fontWeight": 400,
                  "lineHeight": "1.375em",
                },
                "captionNext": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.75rem",
                  "fontWeight": 400,
                  "letterSpacing": "0.03333em",
                  "lineHeight": 1.66,
                },
                "display1": Object {
                  "color": "rgba(0, 0, 0, 0.54)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "2.125rem",
                  "fontWeight": 400,
                  "lineHeight": "1.20588em",
                },
                "display2": Object {
                  "color": "rgba(0, 0, 0, 0.54)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "2.8125rem",
                  "fontWeight": 400,
                  "lineHeight": "1.13333em",
                  "marginLeft": "-.02em",
                },
                "display3": Object {
                  "color": "rgba(0, 0, 0, 0.54)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "3.5rem",
                  "fontWeight": 400,
                  "letterSpacing": "-.02em",
                  "lineHeight": "1.30357em",
                  "marginLeft": "-.02em",
                },
                "display4": Object {
                  "color": "rgba(0, 0, 0, 0.54)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "7rem",
                  "fontWeight": 300,
                  "letterSpacing": "-.04em",
                  "lineHeight": "1.14286em",
                  "marginLeft": "-.04em",
                },
                "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                "fontSize": 14,
                "fontWeightLight": 300,
                "fontWeightMedium": 500,
                "fontWeightRegular": 400,
                "h1": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "6rem",
                  "fontWeight": 300,
                  "letterSpacing": "-0.01562em",
                  "lineHeight": 1,
                },
                "h2": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "3.75rem",
                  "fontWeight": 300,
                  "letterSpacing": "-0.00833em",
                  "lineHeight": 1,
                },
                "h3": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "3rem",
                  "fontWeight": 400,
                  "letterSpacing": "0em",
                  "lineHeight": 1.04,
                },
                "h4": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "2.125rem",
                  "fontWeight": 400,
                  "letterSpacing": "0.00735em",
                  "lineHeight": 1.17,
                },
                "h5": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1.5rem",
                  "fontWeight": 400,
                  "letterSpacing": "0em",
                  "lineHeight": 1.33,
                },
                "h6": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1.25rem",
                  "fontWeight": 500,
                  "letterSpacing": "0.0075em",
                  "lineHeight": 1.6,
                },
                "headline": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1.5rem",
                  "fontWeight": 400,
                  "lineHeight": "1.35417em",
                },
                "overline": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.75rem",
                  "fontWeight": 400,
                  "letterSpacing": "0.08333em",
                  "lineHeight": 2.66,
                  "textTransform": "uppercase",
                },
                "pxToRem": [Function],
                "round": [Function],
                "subheading": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1rem",
                  "fontWeight": 400,
                  "lineHeight": "1.5em",
                },
                "subtitle1": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1rem",
                  "fontWeight": 400,
                  "letterSpacing": "0.00938em",
                  "lineHeight": 1.75,
                },
                "subtitle2": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.875rem",
                  "fontWeight": 500,
                  "letterSpacing": "0.00714em",
                  "lineHeight": 1.57,
                },
                "title": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1.3125rem",
                  "fontWeight": 500,
                  "lineHeight": "1.16667em",
                },
                "useNextVariants": false,
              },
              "zIndex": Object {
                "appBar": 1100,
                "drawer": 1200,
                "mobileStepper": 1000,
                "modal": 1300,
                "snackbar": 1400,
                "tooltip": 1500,
              },
            }
          }
          title={
            <div>
              <div>
                Run was manually terminated
              </div>
            </div>
          }
        >
          <span
            style={
              Object {
                "height": 18,
              }
            }
          >
            <StatusRunning
              style={
                Object {
                  "color": "#80868b",
                  "height": 18,
                  "width": 18,
                }
              }
            />
          </span>
        </Tooltip>
      `);
    });

    it('handles OMITTED phase', () => {
      const tree = shallow(statusToIcon(NodePhase.OMITTED));
      expect(tree).toMatchInlineSnapshot(`
        <Tooltip
          TransitionComponent={[Function]}
          classes={
            Object {
              "popper": "MuiTooltip-popper-1",
              "popperInteractive": "MuiTooltip-popperInteractive-2",
              "tooltip": "MuiTooltip-tooltip-3",
              "tooltipPlacementBottom": "MuiTooltip-tooltipPlacementBottom-8",
              "tooltipPlacementLeft": "MuiTooltip-tooltipPlacementLeft-5",
              "tooltipPlacementRight": "MuiTooltip-tooltipPlacementRight-6",
              "tooltipPlacementTop": "MuiTooltip-tooltipPlacementTop-7",
              "touch": "MuiTooltip-touch-4",
            }
          }
          disableFocusListener={false}
          disableHoverListener={false}
          disableTouchListener={false}
          enterDelay={0}
          enterTouchDelay={1000}
          interactive={false}
          leaveDelay={0}
          leaveTouchDelay={1500}
          placement="bottom"
          theme={
            Object {
              "breakpoints": Object {
                "between": [Function],
                "down": [Function],
                "keys": Array [
                  "xs",
                  "sm",
                  "md",
                  "lg",
                  "xl",
                ],
                "only": [Function],
                "up": [Function],
                "values": Object {
                  "lg": 1280,
                  "md": 960,
                  "sm": 600,
                  "xl": 1920,
                  "xs": 0,
                },
                "width": [Function],
              },
              "direction": "ltr",
              "mixins": Object {
                "gutters": [Function],
                "toolbar": Object {
                  "@media (min-width:0px) and (orientation: landscape)": Object {
                    "minHeight": 48,
                  },
                  "@media (min-width:600px)": Object {
                    "minHeight": 64,
                  },
                  "minHeight": 56,
                },
              },
              "overrides": Object {},
              "palette": Object {
                "action": Object {
                  "active": "rgba(0, 0, 0, 0.54)",
                  "disabled": "rgba(0, 0, 0, 0.26)",
                  "disabledBackground": "rgba(0, 0, 0, 0.12)",
                  "hover": "rgba(0, 0, 0, 0.08)",
                  "hoverOpacity": 0.08,
                  "selected": "rgba(0, 0, 0, 0.14)",
                },
                "augmentColor": [Function],
                "background": Object {
                  "default": "#fafafa",
                  "paper": "#fff",
                },
                "common": Object {
                  "black": "#000",
                  "white": "#fff",
                },
                "contrastThreshold": 3,
                "divider": "rgba(0, 0, 0, 0.12)",
                "error": Object {
                  "contrastText": "#fff",
                  "dark": "#d32f2f",
                  "light": "#e57373",
                  "main": "#f44336",
                },
                "getContrastText": [Function],
                "grey": Object {
                  "100": "#f5f5f5",
                  "200": "#eeeeee",
                  "300": "#e0e0e0",
                  "400": "#bdbdbd",
                  "50": "#fafafa",
                  "500": "#9e9e9e",
                  "600": "#757575",
                  "700": "#616161",
                  "800": "#424242",
                  "900": "#212121",
                  "A100": "#d5d5d5",
                  "A200": "#aaaaaa",
                  "A400": "#303030",
                  "A700": "#616161",
                },
                "primary": Object {
                  "contrastText": "#fff",
                  "dark": "#303f9f",
                  "light": "#7986cb",
                  "main": "#3f51b5",
                },
                "secondary": Object {
                  "contrastText": "#fff",
                  "dark": "#c51162",
                  "light": "#ff4081",
                  "main": "#f50057",
                },
                "text": Object {
                  "disabled": "rgba(0, 0, 0, 0.38)",
                  "hint": "rgba(0, 0, 0, 0.38)",
                  "primary": "rgba(0, 0, 0, 0.87)",
                  "secondary": "rgba(0, 0, 0, 0.54)",
                },
                "tonalOffset": 0.2,
                "type": "light",
              },
              "props": Object {},
              "shadows": Array [
                "none",
                "0px 1px 3px 0px rgba(0,0,0,0.2),0px 1px 1px 0px rgba(0,0,0,0.14),0px 2px 1px -1px rgba(0,0,0,0.12)",
                "0px 1px 5px 0px rgba(0,0,0,0.2),0px 2px 2px 0px rgba(0,0,0,0.14),0px 3px 1px -2px rgba(0,0,0,0.12)",
                "0px 1px 8px 0px rgba(0,0,0,0.2),0px 3px 4px 0px rgba(0,0,0,0.14),0px 3px 3px -2px rgba(0,0,0,0.12)",
                "0px 2px 4px -1px rgba(0,0,0,0.2),0px 4px 5px 0px rgba(0,0,0,0.14),0px 1px 10px 0px rgba(0,0,0,0.12)",
                "0px 3px 5px -1px rgba(0,0,0,0.2),0px 5px 8px 0px rgba(0,0,0,0.14),0px 1px 14px 0px rgba(0,0,0,0.12)",
                "0px 3px 5px -1px rgba(0,0,0,0.2),0px 6px 10px 0px rgba(0,0,0,0.14),0px 1px 18px 0px rgba(0,0,0,0.12)",
                "0px 4px 5px -2px rgba(0,0,0,0.2),0px 7px 10px 1px rgba(0,0,0,0.14),0px 2px 16px 1px rgba(0,0,0,0.12)",
                "0px 5px 5px -3px rgba(0,0,0,0.2),0px 8px 10px 1px rgba(0,0,0,0.14),0px 3px 14px 2px rgba(0,0,0,0.12)",
                "0px 5px 6px -3px rgba(0,0,0,0.2),0px 9px 12px 1px rgba(0,0,0,0.14),0px 3px 16px 2px rgba(0,0,0,0.12)",
                "0px 6px 6px -3px rgba(0,0,0,0.2),0px 10px 14px 1px rgba(0,0,0,0.14),0px 4px 18px 3px rgba(0,0,0,0.12)",
                "0px 6px 7px -4px rgba(0,0,0,0.2),0px 11px 15px 1px rgba(0,0,0,0.14),0px 4px 20px 3px rgba(0,0,0,0.12)",
                "0px 7px 8px -4px rgba(0,0,0,0.2),0px 12px 17px 2px rgba(0,0,0,0.14),0px 5px 22px 4px rgba(0,0,0,0.12)",
                "0px 7px 8px -4px rgba(0,0,0,0.2),0px 13px 19px 2px rgba(0,0,0,0.14),0px 5px 24px 4px rgba(0,0,0,0.12)",
                "0px 7px 9px -4px rgba(0,0,0,0.2),0px 14px 21px 2px rgba(0,0,0,0.14),0px 5px 26px 4px rgba(0,0,0,0.12)",
                "0px 8px 9px -5px rgba(0,0,0,0.2),0px 15px 22px 2px rgba(0,0,0,0.14),0px 6px 28px 5px rgba(0,0,0,0.12)",
                "0px 8px 10px -5px rgba(0,0,0,0.2),0px 16px 24px 2px rgba(0,0,0,0.14),0px 6px 30px 5px rgba(0,0,0,0.12)",
                "0px 8px 11px -5px rgba(0,0,0,0.2),0px 17px 26px 2px rgba(0,0,0,0.14),0px 6px 32px 5px rgba(0,0,0,0.12)",
                "0px 9px 11px -5px rgba(0,0,0,0.2),0px 18px 28px 2px rgba(0,0,0,0.14),0px 7px 34px 6px rgba(0,0,0,0.12)",
                "0px 9px 12px -6px rgba(0,0,0,0.2),0px 19px 29px 2px rgba(0,0,0,0.14),0px 7px 36px 6px rgba(0,0,0,0.12)",
                "0px 10px 13px -6px rgba(0,0,0,0.2),0px 20px 31px 3px rgba(0,0,0,0.14),0px 8px 38px 7px rgba(0,0,0,0.12)",
                "0px 10px 13px -6px rgba(0,0,0,0.2),0px 21px 33px 3px rgba(0,0,0,0.14),0px 8px 40px 7px rgba(0,0,0,0.12)",
                "0px 10px 14px -6px rgba(0,0,0,0.2),0px 22px 35px 3px rgba(0,0,0,0.14),0px 8px 42px 7px rgba(0,0,0,0.12)",
                "0px 11px 14px -7px rgba(0,0,0,0.2),0px 23px 36px 3px rgba(0,0,0,0.14),0px 9px 44px 8px rgba(0,0,0,0.12)",
                "0px 11px 15px -7px rgba(0,0,0,0.2),0px 24px 38px 3px rgba(0,0,0,0.14),0px 9px 46px 8px rgba(0,0,0,0.12)",
              ],
              "shape": Object {
                "borderRadius": 4,
              },
              "spacing": Object {
                "unit": 8,
              },
              "transitions": Object {
                "create": [Function],
                "duration": Object {
                  "complex": 375,
                  "enteringScreen": 225,
                  "leavingScreen": 195,
                  "short": 250,
                  "shorter": 200,
                  "shortest": 150,
                  "standard": 300,
                },
                "easing": Object {
                  "easeIn": "cubic-bezier(0.4, 0, 1, 1)",
                  "easeInOut": "cubic-bezier(0.4, 0, 0.2, 1)",
                  "easeOut": "cubic-bezier(0.0, 0, 0.2, 1)",
                  "sharp": "cubic-bezier(0.4, 0, 0.6, 1)",
                },
                "getAutoHeightDuration": [Function],
              },
              "typography": Object {
                "body1": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.875rem",
                  "fontWeight": 400,
                  "lineHeight": "1.46429em",
                },
                "body1Next": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1rem",
                  "fontWeight": 400,
                  "letterSpacing": "0.00938em",
                  "lineHeight": 1.5,
                },
                "body2": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.875rem",
                  "fontWeight": 500,
                  "lineHeight": "1.71429em",
                },
                "body2Next": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.875rem",
                  "fontWeight": 400,
                  "letterSpacing": "0.01071em",
                  "lineHeight": 1.5,
                },
                "button": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.875rem",
                  "fontWeight": 500,
                  "textTransform": "uppercase",
                },
                "buttonNext": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.875rem",
                  "fontWeight": 500,
                  "letterSpacing": "0.02857em",
                  "lineHeight": 1.5,
                  "textTransform": "uppercase",
                },
                "caption": Object {
                  "color": "rgba(0, 0, 0, 0.54)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.75rem",
                  "fontWeight": 400,
                  "lineHeight": "1.375em",
                },
                "captionNext": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.75rem",
                  "fontWeight": 400,
                  "letterSpacing": "0.03333em",
                  "lineHeight": 1.66,
                },
                "display1": Object {
                  "color": "rgba(0, 0, 0, 0.54)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "2.125rem",
                  "fontWeight": 400,
                  "lineHeight": "1.20588em",
                },
                "display2": Object {
                  "color": "rgba(0, 0, 0, 0.54)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "2.8125rem",
                  "fontWeight": 400,
                  "lineHeight": "1.13333em",
                  "marginLeft": "-.02em",
                },
                "display3": Object {
                  "color": "rgba(0, 0, 0, 0.54)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "3.5rem",
                  "fontWeight": 400,
                  "letterSpacing": "-.02em",
                  "lineHeight": "1.30357em",
                  "marginLeft": "-.02em",
                },
                "display4": Object {
                  "color": "rgba(0, 0, 0, 0.54)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "7rem",
                  "fontWeight": 300,
                  "letterSpacing": "-.04em",
                  "lineHeight": "1.14286em",
                  "marginLeft": "-.04em",
                },
                "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                "fontSize": 14,
                "fontWeightLight": 300,
                "fontWeightMedium": 500,
                "fontWeightRegular": 400,
                "h1": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "6rem",
                  "fontWeight": 300,
                  "letterSpacing": "-0.01562em",
                  "lineHeight": 1,
                },
                "h2": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "3.75rem",
                  "fontWeight": 300,
                  "letterSpacing": "-0.00833em",
                  "lineHeight": 1,
                },
                "h3": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "3rem",
                  "fontWeight": 400,
                  "letterSpacing": "0em",
                  "lineHeight": 1.04,
                },
                "h4": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "2.125rem",
                  "fontWeight": 400,
                  "letterSpacing": "0.00735em",
                  "lineHeight": 1.17,
                },
                "h5": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1.5rem",
                  "fontWeight": 400,
                  "letterSpacing": "0em",
                  "lineHeight": 1.33,
                },
                "h6": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1.25rem",
                  "fontWeight": 500,
                  "letterSpacing": "0.0075em",
                  "lineHeight": 1.6,
                },
                "headline": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1.5rem",
                  "fontWeight": 400,
                  "lineHeight": "1.35417em",
                },
                "overline": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.75rem",
                  "fontWeight": 400,
                  "letterSpacing": "0.08333em",
                  "lineHeight": 2.66,
                  "textTransform": "uppercase",
                },
                "pxToRem": [Function],
                "round": [Function],
                "subheading": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1rem",
                  "fontWeight": 400,
                  "lineHeight": "1.5em",
                },
                "subtitle1": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1rem",
                  "fontWeight": 400,
                  "letterSpacing": "0.00938em",
                  "lineHeight": 1.75,
                },
                "subtitle2": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "0.875rem",
                  "fontWeight": 500,
                  "letterSpacing": "0.00714em",
                  "lineHeight": 1.57,
                },
                "title": Object {
                  "color": "rgba(0, 0, 0, 0.87)",
                  "fontFamily": "\\"Roboto\\", \\"Helvetica\\", \\"Arial\\", sans-serif",
                  "fontSize": "1.3125rem",
                  "fontWeight": 500,
                  "lineHeight": "1.16667em",
                },
                "useNextVariants": false,
              },
              "zIndex": Object {
                "appBar": 1100,
                "drawer": 1200,
                "mobileStepper": 1000,
                "modal": 1300,
                "snackbar": 1400,
                "tooltip": 1500,
              },
            }
          }
          title={
            <div>
              <div>
                Run was omitted because the previous step failed.
              </div>
            </div>
          }
        >
          <span
            style={
              Object {
                "height": 18,
              }
            }
          >
            <pure(BlockIcon)
              style={
                Object {
                  "color": "#5f6368",
                  "height": 18,
                  "width": 18,
                }
              }
            />
          </span>
        </Tooltip>
      `);
    });

    it('displays start and end dates if both are provided', () => {
      const tree = shallow(statusToIcon(NodePhase.SUCCEEDED, startDate, endDate));
      expect(tree).toMatchSnapshot();
    });

    it('does not display a end date if none was provided', () => {
      const tree = shallow(statusToIcon(NodePhase.SUCCEEDED, startDate));
      expect(tree).toMatchSnapshot();
    });

    it('does not display a start date if none was provided', () => {
      const tree = shallow(statusToIcon(NodePhase.SUCCEEDED, undefined, endDate));
      expect(tree).toMatchSnapshot();
    });

    it('does not display any dates if neither was provided', () => {
      const tree = shallow(statusToIcon(NodePhase.SUCCEEDED /* No dates */));
      expect(tree).toMatchSnapshot();
    });

    Object.keys(NodePhase).map(status =>
      it('renders an icon with tooltip for phase: ' + status, () => {
        const tree = shallow(statusToIcon(NodePhase[status]));
        expect(tree).toMatchSnapshot();
      }),
    );
  });
});
