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

import { URLParser, useURLParser } from './URLParser';
import { renderHook, act } from '@testing-library/react';

// Mock React Router for hook tests
const mockNavigate = jest.fn();
const mockLocation = { search: '?testkey=testvalue' };

jest.mock('react-router-dom', () => ({
  ...jest.requireActual('react-router-dom'),
  useLocation: () => mockLocation,
  useNavigate: () => mockNavigate,
}));

describe('useURLParser hook', () => {
  beforeEach(() => {
    mockNavigate.mockClear();
    mockLocation.search = '?testkey=testvalue';
  });

  it('gets query string param by name', () => {
    const { result } = renderHook(() => useURLParser());
    const value = result.current.get('testkey' as any);
    expect(value).toEqual('testvalue');
  });

  it('returns null when getting a nonexistent query string param', () => {
    const { result } = renderHook(() => useURLParser());
    expect(result.current.get('nonexistent' as any)).toBeNull();
  });

  it('gets query string param by name where multiple keys defined', () => {
    mockLocation.search = '?testkey=testvalue&k2=v2';
    const { result } = renderHook(() => useURLParser());
    const value = result.current.get('testkey' as any);
    expect(value).toEqual('testvalue');
  });

  it('sets query string param by name', () => {
    const { result } = renderHook(() => useURLParser());
    act(() => {
      result.current.set('testkey' as any, 'newvalue');
    });
    expect(mockNavigate).toHaveBeenCalledWith(
      { search: 'testkey=newvalue' },
      { replace: true }
    );
  });

  it('creates new query string param when using set if not exists', () => {
    const { result } = renderHook(() => useURLParser());
    act(() => {
      result.current.set('newkey' as any, 'newvalue');
    });
    expect(mockNavigate).toHaveBeenCalledWith(
      expect.objectContaining({ search: expect.stringContaining('newkey=newvalue') }),
      { replace: true }
    );
  });

  it('removes the query string param when setting it to empty value', () => {
    mockLocation.search = '?testkey=testvalue&otherkey=othervalue';
    const { result } = renderHook(() => useURLParser());
    act(() => {
      result.current.set('testkey' as any, '');
    });
    expect(mockNavigate).toHaveBeenCalledWith(
      { search: 'otherkey=othervalue' },
      { replace: true }
    );
  });

  it('navigates with replace:false when using push', () => {
    const { result } = renderHook(() => useURLParser());
    act(() => {
      result.current.push('testkey' as any, 'newvalue');
    });
    expect(mockNavigate).toHaveBeenCalledWith(
      { search: 'testkey=newvalue' },
      { replace: false }
    );
  });

  it('can build a search string', () => {
    const { result } = renderHook(() => useURLParser());
    expect(result.current.build({ key1: 'value1', key2: 'value2' })).toEqual(
      '?key1=value1&key2=value2',
    );
  });

  it('returns empty query string when given an empty object or no object', () => {
    const { result } = renderHook(() => useURLParser());
    expect(result.current.build()).toEqual('?');
    expect(result.current.build({})).toEqual('?');
  });

  it('clears query string param', () => {
    mockLocation.search = '?testkey=testvalue&otherkey=othervalue';
    const { result } = renderHook(() => useURLParser());
    act(() => {
      result.current.clear('testkey' as any);
    });
    expect(mockNavigate).toHaveBeenCalledWith(
      { search: 'otherkey=othervalue' },
      { replace: true }
    );
  });
});

// Legacy URLParser class tests - maintained for backward compatibility
describe('URLParser (Legacy Class)', () => {
  const mockNavigate = jest.fn();
  let mockLocation: { pathname: string; search: string };
  let routerProps: any;

  beforeEach(() => {
    mockNavigate.mockClear();
    mockLocation = { pathname: '/test', search: '' };
    routerProps = { location: mockLocation, navigate: mockNavigate };
  });

  it('gets query string param by name', () => {
    mockLocation.search = '?searchkey=searchvalue';
    routerProps = { location: mockLocation, navigate: mockNavigate };

    const params = new URLParser(routerProps).get('searchkey' as any);
    expect(params).toEqual('searchvalue');
  });

  it('returns null when getting a nonexistent query string param', () => {
    const urlParser = new URLParser(routerProps);
    expect(urlParser.get('nonexistent' as any)).toBeNull();
  });

  it('gets query string param by name where multiple keys defined', () => {
    mockLocation.search = '?searchkey=searchvalue&k2=v2';
    routerProps = { location: mockLocation, navigate: mockNavigate };

    const params = new URLParser(routerProps).get('searchkey' as any);
    expect(params).toEqual('searchvalue');
  });

  it('sets query string param by name', () => {
    mockLocation.search = '?searchkey=searchvalue';
    routerProps = { location: mockLocation, navigate: mockNavigate };

    new URLParser(routerProps).set('searchkey' as any, 'newvalue');
    expect(mockNavigate).toHaveBeenCalledWith(
      { search: 'searchkey=newvalue' },
      { replace: true }
    );
  });

  it('creates new query string param when using set if not exists', () => {
    mockLocation.search = '?searchkey=searchvalue';
    routerProps = { location: mockLocation, navigate: mockNavigate };

    new URLParser(routerProps).set('searchkey2' as any, 'searchvalue2');
    expect(mockNavigate).toHaveBeenCalledWith(
      { search: 'searchkey=searchvalue&searchkey2=searchvalue2' },
      { replace: true }
    );
  });

  it('removes the query string param when setting it to empty value', () => {
    mockLocation.search = '?searchkey=searchvalue';
    routerProps = { location: mockLocation, navigate: mockNavigate };

    new URLParser(routerProps).set('searchkey' as any, '');
    expect(mockNavigate).toHaveBeenCalledWith(
      { search: '' },
      { replace: true }
    );
  });

  it('does not touch other query string params when setting one', () => {
    mockLocation.search = '?searchkey=searchvalue&k2=v2';
    routerProps = { location: mockLocation, navigate: mockNavigate };

    new URLParser(routerProps).set('searchkey' as any, 'newvalue');
    expect(mockNavigate).toHaveBeenCalledWith(
      { search: 'searchkey=newvalue&k2=v2' },
      { replace: true }
    );
  });

  it('does not touch other query string params when clearing one', () => {
    mockLocation.search = '?searchkey=searchvalue&k2=v2';
    routerProps = { location: mockLocation, navigate: mockNavigate };

    new URLParser(routerProps).clear('searchkey' as any);
    expect(mockNavigate).toHaveBeenCalledWith(
      { search: 'k2=v2' },
      { replace: true }
    );
  });

  it('sets query string param when using push', () => {
    mockLocation.search = '?searchkey=searchvalue&k2=v2';
    routerProps = { location: mockLocation, navigate: mockNavigate };

    new URLParser(routerProps).push('searchkey' as any, 'newvalue');
    expect(mockNavigate).toHaveBeenCalledWith(
      { search: 'searchkey=newvalue&k2=v2' },
      { replace: false }
    );
  });

  it('removes query string param when push an empty value', () => {
    mockLocation.search = '?searchkey=searchvalue&k2=v2';
    routerProps = { location: mockLocation, navigate: mockNavigate };

    new URLParser(routerProps).push('searchkey' as any, '');
    expect(mockNavigate).toHaveBeenCalledWith(
      { search: 'k2=v2' },
      { replace: false }
    );
  });

  it('can build a search string', () => {
    const urlParser = new URLParser(routerProps);
    expect(urlParser.build({ key1: 'value1', key2: 'value2' })).toEqual(
      '?key1=value1&key2=value2',
    );
  });

  it('returns empty query string when given an empty object or no object', () => {
    const urlParser = new URLParser(routerProps);
    expect(urlParser.build()).toEqual('?');
    expect(urlParser.build({})).toEqual('?');
  });
});


