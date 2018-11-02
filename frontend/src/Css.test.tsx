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

import { spacing, _paddingInternal } from './Css';
import * as Css from './Css';

describe('Css', () => {
  describe('padding', () => {
    it('returns padding units in all directions by default', () => {
      expect(_paddingInternal()).toEqual({
        paddingBottom: spacing.base,
        paddingLeft: spacing.base,
        paddingRight: spacing.base,
        paddingTop: spacing.base,
      });
    });

    it('returns specified padding units in all directions', () => {
      expect(_paddingInternal(100)).toEqual({
        paddingBottom: 100,
        paddingLeft: 100,
        paddingRight: 100,
        paddingTop: 100,
      });
    });

    it('returns default units in specified directions', () => {
      expect(_paddingInternal(undefined, 'lr')).toEqual({
        paddingLeft: spacing.base,
        paddingRight: spacing.base,
      });
    });

    it('calls internal padding with the same arguments', () => {
      const spy = jest.spyOn(Css, 'padding');
      Css.padding(123, 'abcdefg');
      expect(spy).toHaveBeenCalledWith(123, 'abcdefg');
    });
  });
});
