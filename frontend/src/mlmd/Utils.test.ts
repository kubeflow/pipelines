/*
 * Copyright 2018,2021 The Kubeflow Authors
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

import { doubleValue, intValue, stringValue } from './TestUtils';
import { Artifact, Value } from 'src/third_party/mlmd';
import { getMetadataValue, getResourceProperty } from './Utils';

describe('Utils', () => {
  describe('getResourceProperty', () => {
    it('returns null if resource has no properties', () => {
      expect(getResourceProperty(new Artifact(), 'testPropName')).toBeNull();
    });

    it('returns null if resource has no custom properties', () => {
      expect(getResourceProperty(new Artifact(), 'testCustomPropName', true)).toBeNull();
    });

    it('returns null if resource has no property with the provided name', () => {
      const resource = new Artifact();
      resource.getPropertiesMap().set('somePropName', doubleValue(123));
      expect(getResourceProperty(resource, 'testPropName')).toBeNull();
    });

    it('returns if resource has no property with specified name if fromCustomProperties is false', () => {
      const resource = new Artifact();
      resource.getCustomPropertiesMap().set('testCustomPropName', doubleValue(123));
      expect(
        getResourceProperty(resource, 'testCustomPropName', /* fromCustomProperties= */ false),
      ).toBeNull();
    });

    it('returns if resource has no custom property with specified name if fromCustomProperties is true', () => {
      const resource = new Artifact();
      resource.getPropertiesMap().set('testPropName', doubleValue(123));
      expect(
        getResourceProperty(resource, 'testPropName', /* fromCustomProperties= */ true),
      ).toBeNull();
    });

    it('returns the value of the property with the provided name', () => {
      const resource = new Artifact();
      resource.getPropertiesMap().set('testPropName', doubleValue(123));
      expect(getResourceProperty(resource, 'testPropName')).toEqual(123);
    });

    it('returns the value of the custom property with the provided name', () => {
      const resource = new Artifact();
      resource.getCustomPropertiesMap().set('testCustomPropName', stringValue('abc'));
      expect(
        getResourceProperty(resource, 'testCustomPropName', /* fromCustomProperties= */ true),
      ).toEqual('abc');
    });
  });

  describe('getMetadataValue', () => {
    it('returns a value of type double', () => {
      expect(getMetadataValue(doubleValue(123))).toEqual(123);
    });

    it('returns a value of type int', () => {
      expect(getMetadataValue(intValue(123))).toEqual(123);
    });

    it('returns a value of type string', () => {
      expect(getMetadataValue(stringValue('abc'))).toEqual('abc');
    });

    it('returns an empty string if Value has no value', () => {
      expect(getMetadataValue(new Value())).toEqual('');
    });
  });
});
