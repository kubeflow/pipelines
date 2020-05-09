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

import DetailsTable from './DetailsTable';
import { render } from '@testing-library/react';

describe('DetailsTable', () => {
  it('shows no rows', () => {
    const { container } = render(<DetailsTable fields={[]} />);
    expect(container).toMatchInlineSnapshot(`
      <div>
        <div />
      </div>
    `);
  });

  it('shows one row', () => {
    const { container } = render(<DetailsTable fields={[['key', 'value']]} />);
    expect(container).toMatchInlineSnapshot(`
      <div>
        <div>
          <div
            class="row"
          >
            <span
              class="key"
            >
              key
            </span>
            <span
              class="valueText"
            >
              value
            </span>
          </div>
        </div>
      </div>
    `);
  });

  it('shows a row with a title', () => {
    const { container } = render(<DetailsTable title='some title' fields={[['key', 'value']]} />);
    expect(container).toMatchInlineSnapshot(`
      <div>
        <div
          class="header"
        >
          some title
        </div>
        <div>
          <div
            class="row"
          >
            <span
              class="key"
            >
              key
            </span>
            <span
              class="valueText"
            >
              value
            </span>
          </div>
        </div>
      </div>
    `);
  });

  it('shows key and value for large values', () => {
    const { container } = render(
      <DetailsTable
        fields={[
          [
            'key',
            `Lorem Ipsum is simply dummy text of the printing and typesetting
      industry. Lorem Ipsum has been the industry's standard dummy text ever
      since the 1500s, when an unknown printer took a galley of type and
      scrambled it to make a type specimen book. It has survived not only five
      centuries, but also the leap into electronic typesetting, remaining
      essentially unchanged. It was popularised in the 1960s with the release
      of Letraset sheets containing Lorem Ipsum passages, and more recently
      with desktop publishing software like Aldus PageMaker including versions
      of Lorem Ipsum.`,
          ],
        ]}
      />,
    );
    expect(container).toMatchInlineSnapshot(`
      <div>
        <div>
          <div
            class="row"
          >
            <span
              class="key"
            >
              key
            </span>
            <span
              class="valueText"
            >
              Lorem Ipsum is simply dummy text of the printing and typesetting
            industry. Lorem Ipsum has been the industry's standard dummy text ever
            since the 1500s, when an unknown printer took a galley of type and
            scrambled it to make a type specimen book. It has survived not only five
            centuries, but also the leap into electronic typesetting, remaining
            essentially unchanged. It was popularised in the 1960s with the release
            of Letraset sheets containing Lorem Ipsum passages, and more recently
            with desktop publishing software like Aldus PageMaker including versions
            of Lorem Ipsum.
            </span>
          </div>
        </div>
      </div>
    `);
  });

  it('shows key and value in row', () => {
    const { container } = render(<DetailsTable fields={[['key', 'value']]} />);
    expect(container).toMatchInlineSnapshot(`
      <div>
        <div>
          <div
            class="row"
          >
            <span
              class="key"
            >
              key
            </span>
            <span
              class="valueText"
            >
              value
            </span>
          </div>
        </div>
      </div>
    `);
  });

  it('shows key and JSON value in row', () => {
    const { container } = render(
      <DetailsTable fields={[['key', JSON.stringify([{ jsonKey: 'jsonValue' }])]]} />,
    );
    expect(container).toMatchInlineSnapshot(`
      <div>
        <div>
          <div
            class="row"
          >
            <span
              class="key"
            >
              key
            </span>
            <div
              class=" ace_editor ace-github"
              id="brace-editor"
              style="width: 100%; height: 500px;"
            >
              <textarea
                autocapitalize="off"
                autocorrect="off"
                class="ace_text-input"
                spellcheck="false"
                style="opacity: 0;"
                wrap="off"
              />
              <div
                aria-hidden="true"
                class="ace_gutter"
              >
                <div
                  class="ace_layer ace_gutter-layer ace_folding-enabled"
                />
                <div
                  class="ace_gutter-active-line"
                />
              </div>
              <div
                class="ace_scroller"
              >
                <div
                  class="ace_content"
                >
                  <div
                    class="ace_layer ace_print-margin-layer"
                  >
                    <div
                      class="ace_print-margin"
                      style="left: 4px; visibility: visible;"
                    />
                  </div>
                  <div
                    class="ace_layer ace_marker-layer"
                  />
                  <div
                    class="ace_layer ace_text-layer"
                    style="padding: 0px 4px;"
                  />
                  <div
                    class="ace_layer ace_marker-layer"
                  />
                  <div
                    class="ace_layer ace_cursor-layer ace_hidden-cursors"
                  >
                    <div
                      class="ace_cursor"
                    />
                  </div>
                </div>
              </div>
              <div
                class="ace_scrollbar ace_scrollbar-v"
                style="display: none; width: 20px;"
              >
                <div
                  class="ace_scrollbar-inner"
                  style="width: 20px;"
                />
              </div>
              <div
                class="ace_scrollbar ace_scrollbar-h"
                style="display: none; height: 20px;"
              >
                <div
                  class="ace_scrollbar-inner"
                  style="height: 20px;"
                />
              </div>
              <div
                style="height: auto; width: auto; top: 0px; left: 0px; visibility: hidden; position: absolute; white-space: pre; overflow: hidden;"
              >
                <div
                  style="height: auto; width: auto; top: 0px; left: 0px; visibility: hidden; position: absolute; white-space: pre; overflow: visible;"
                />
                <div
                  style="height: auto; width: auto; top: 0px; left: 0px; visibility: hidden; position: absolute; white-space: pre; overflow: visible;"
                >
                  XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    `);
  });

  it('does render arrays as JSON', () => {
    const { container } = render(<DetailsTable fields={[['key', '[]']]} />);
    expect(container).toMatchInlineSnapshot(`
      <div>
        <div>
          <div
            class="row"
          >
            <span
              class="key"
            >
              key
            </span>
            <div
              class=" ace_editor ace-github"
              id="brace-editor"
              style="width: 100%; height: 500px;"
            >
              <textarea
                autocapitalize="off"
                autocorrect="off"
                class="ace_text-input"
                spellcheck="false"
                style="opacity: 0;"
                wrap="off"
              />
              <div
                aria-hidden="true"
                class="ace_gutter"
              >
                <div
                  class="ace_layer ace_gutter-layer ace_folding-enabled"
                />
                <div
                  class="ace_gutter-active-line"
                />
              </div>
              <div
                class="ace_scroller"
              >
                <div
                  class="ace_content"
                >
                  <div
                    class="ace_layer ace_print-margin-layer"
                  >
                    <div
                      class="ace_print-margin"
                      style="left: 4px; visibility: visible;"
                    />
                  </div>
                  <div
                    class="ace_layer ace_marker-layer"
                  />
                  <div
                    class="ace_layer ace_text-layer"
                    style="padding: 0px 4px;"
                  />
                  <div
                    class="ace_layer ace_marker-layer"
                  />
                  <div
                    class="ace_layer ace_cursor-layer ace_hidden-cursors"
                  >
                    <div
                      class="ace_cursor"
                    />
                  </div>
                </div>
              </div>
              <div
                class="ace_scrollbar ace_scrollbar-v"
                style="display: none; width: 20px;"
              >
                <div
                  class="ace_scrollbar-inner"
                  style="width: 20px;"
                />
              </div>
              <div
                class="ace_scrollbar ace_scrollbar-h"
                style="display: none; height: 20px;"
              >
                <div
                  class="ace_scrollbar-inner"
                  style="height: 20px;"
                />
              </div>
              <div
                style="height: auto; width: auto; top: 0px; left: 0px; visibility: hidden; position: absolute; white-space: pre; overflow: hidden;"
              >
                <div
                  style="height: auto; width: auto; top: 0px; left: 0px; visibility: hidden; position: absolute; white-space: pre; overflow: visible;"
                />
                <div
                  style="height: auto; width: auto; top: 0px; left: 0px; visibility: hidden; position: absolute; white-space: pre; overflow: visible;"
                >
                  XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    `);
  });

  it('does render empty object as JSON', () => {
    const { container } = render(<DetailsTable fields={[['key', '{}']]} />);
    expect(container).toMatchInlineSnapshot(`
      <div>
        <div>
          <div
            class="row"
          >
            <span
              class="key"
            >
              key
            </span>
            <div
              class=" ace_editor ace-github"
              id="brace-editor"
              style="width: 100%; height: 500px;"
            >
              <textarea
                autocapitalize="off"
                autocorrect="off"
                class="ace_text-input"
                spellcheck="false"
                style="opacity: 0;"
                wrap="off"
              />
              <div
                aria-hidden="true"
                class="ace_gutter"
              >
                <div
                  class="ace_layer ace_gutter-layer ace_folding-enabled"
                />
                <div
                  class="ace_gutter-active-line"
                />
              </div>
              <div
                class="ace_scroller"
              >
                <div
                  class="ace_content"
                >
                  <div
                    class="ace_layer ace_print-margin-layer"
                  >
                    <div
                      class="ace_print-margin"
                      style="left: 4px; visibility: visible;"
                    />
                  </div>
                  <div
                    class="ace_layer ace_marker-layer"
                  />
                  <div
                    class="ace_layer ace_text-layer"
                    style="padding: 0px 4px;"
                  />
                  <div
                    class="ace_layer ace_marker-layer"
                  />
                  <div
                    class="ace_layer ace_cursor-layer ace_hidden-cursors"
                  >
                    <div
                      class="ace_cursor"
                    />
                  </div>
                </div>
              </div>
              <div
                class="ace_scrollbar ace_scrollbar-v"
                style="display: none; width: 20px;"
              >
                <div
                  class="ace_scrollbar-inner"
                  style="width: 20px;"
                />
              </div>
              <div
                class="ace_scrollbar ace_scrollbar-h"
                style="display: none; height: 20px;"
              >
                <div
                  class="ace_scrollbar-inner"
                  style="height: 20px;"
                />
              </div>
              <div
                style="height: auto; width: auto; top: 0px; left: 0px; visibility: hidden; position: absolute; white-space: pre; overflow: hidden;"
              >
                <div
                  style="height: auto; width: auto; top: 0px; left: 0px; visibility: hidden; position: absolute; white-space: pre; overflow: visible;"
                />
                <div
                  style="height: auto; width: auto; top: 0px; left: 0px; visibility: hidden; position: absolute; white-space: pre; overflow: visible;"
                >
                  XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    `);
  });

  it('does not render nulls as JSON', () => {
    const { container } = render(<DetailsTable fields={[['key', 'null']]} />);
    expect(container).toMatchInlineSnapshot(`
      <div>
        <div>
          <div
            class="row"
          >
            <span
              class="key"
            >
              key
            </span>
            <span
              class="valueText"
            >
              null
            </span>
          </div>
        </div>
      </div>
    `);
  });

  it('does not render numbers as JSON', () => {
    const { container } = render(<DetailsTable fields={[['key', '10']]} />);
    expect(container).toMatchInlineSnapshot(`
      <div>
        <div>
          <div
            class="row"
          >
            <span
              class="key"
            >
              key
            </span>
            <span
              class="valueText"
            >
              10
            </span>
          </div>
        </div>
      </div>
    `);
  });

  it('does not render strings as JSON', () => {
    const { container } = render(<DetailsTable fields={[['key', '"some string"']]} />);
    expect(container).toMatchInlineSnapshot(`
      <div>
        <div>
          <div
            class="row"
          >
            <span
              class="key"
            >
              key
            </span>
            <span
              class="valueText"
            >
              "some string"
            </span>
          </div>
        </div>
      </div>
    `);
  });

  it('does not render booleans as JSON', () => {
    const { container } = render(
      <DetailsTable
        fields={[
          ['key1', 'true'],
          ['key2', 'false'],
        ]}
      />,
    );
    expect(container).toMatchInlineSnapshot(`
      <div>
        <div>
          <div
            class="row"
          >
            <span
              class="key"
            >
              key1
            </span>
            <span
              class="valueText"
            >
              true
            </span>
          </div>
          <div
            class="row"
          >
            <span
              class="key"
            >
              key2
            </span>
            <span
              class="valueText"
            >
              false
            </span>
          </div>
        </div>
      </div>
    `);
  });

  it('shows keys and values for multiple rows', () => {
    const { container } = render(
      <DetailsTable
        fields={[
          ['key1', 'value1'],
          ['key2', JSON.stringify([{ jsonKey: 'jsonValue2' }])],
          ['key3', 'value3'],
          ['key4', 'value4'],
          ['key5', JSON.stringify({ jsonKey: { nestedJsonKey: 'jsonValue' } })],
          ['key6', 'value6'],
          ['key6', 'value7'],
          ['key', { key: 'foobar', bucket: 'bucket', endpoint: 's3.amazonaws.com' }],
        ]}
      />,
    );
    expect(container).toMatchInlineSnapshot(`
      <div>
        <div>
          <div
            class="row"
          >
            <span
              class="key"
            >
              key1
            </span>
            <span
              class="valueText"
            >
              value1
            </span>
          </div>
          <div
            class="row"
          >
            <span
              class="key"
            >
              key2
            </span>
            <div
              class=" ace_editor ace-github"
              id="brace-editor"
              style="width: 100%; height: 500px;"
            >
              <textarea
                autocapitalize="off"
                autocorrect="off"
                class="ace_text-input"
                spellcheck="false"
                style="opacity: 0;"
                wrap="off"
              />
              <div
                aria-hidden="true"
                class="ace_gutter"
              >
                <div
                  class="ace_layer ace_gutter-layer ace_folding-enabled"
                />
                <div
                  class="ace_gutter-active-line"
                />
              </div>
              <div
                class="ace_scroller"
              >
                <div
                  class="ace_content"
                >
                  <div
                    class="ace_layer ace_print-margin-layer"
                  >
                    <div
                      class="ace_print-margin"
                      style="left: 4px; visibility: visible;"
                    />
                  </div>
                  <div
                    class="ace_layer ace_marker-layer"
                  />
                  <div
                    class="ace_layer ace_text-layer"
                    style="padding: 0px 4px;"
                  />
                  <div
                    class="ace_layer ace_marker-layer"
                  />
                  <div
                    class="ace_layer ace_cursor-layer ace_hidden-cursors"
                  >
                    <div
                      class="ace_cursor"
                    />
                  </div>
                </div>
              </div>
              <div
                class="ace_scrollbar ace_scrollbar-v"
                style="display: none; width: 20px;"
              >
                <div
                  class="ace_scrollbar-inner"
                  style="width: 20px;"
                />
              </div>
              <div
                class="ace_scrollbar ace_scrollbar-h"
                style="display: none; height: 20px;"
              >
                <div
                  class="ace_scrollbar-inner"
                  style="height: 20px;"
                />
              </div>
              <div
                style="height: auto; width: auto; top: 0px; left: 0px; visibility: hidden; position: absolute; white-space: pre; overflow: hidden;"
              >
                <div
                  style="height: auto; width: auto; top: 0px; left: 0px; visibility: hidden; position: absolute; white-space: pre; overflow: visible;"
                />
                <div
                  style="height: auto; width: auto; top: 0px; left: 0px; visibility: hidden; position: absolute; white-space: pre; overflow: visible;"
                >
                  XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
                </div>
              </div>
            </div>
          </div>
          <div
            class="row"
          >
            <span
              class="key"
            >
              key3
            </span>
            <span
              class="valueText"
            >
              value3
            </span>
          </div>
          <div
            class="row"
          >
            <span
              class="key"
            >
              key4
            </span>
            <span
              class="valueText"
            >
              value4
            </span>
          </div>
          <div
            class="row"
          >
            <span
              class="key"
            >
              key5
            </span>
            <div
              class=" ace_editor ace-github"
              id="brace-editor"
              style="width: 100%; height: 500px;"
            >
              <textarea
                autocapitalize="off"
                autocorrect="off"
                class="ace_text-input"
                spellcheck="false"
                style="opacity: 0;"
                wrap="off"
              />
              <div
                aria-hidden="true"
                class="ace_gutter"
              >
                <div
                  class="ace_layer ace_gutter-layer ace_folding-enabled"
                />
                <div
                  class="ace_gutter-active-line"
                />
              </div>
              <div
                class="ace_scroller"
              >
                <div
                  class="ace_content"
                >
                  <div
                    class="ace_layer ace_print-margin-layer"
                  >
                    <div
                      class="ace_print-margin"
                      style="left: 4px; visibility: visible;"
                    />
                  </div>
                  <div
                    class="ace_layer ace_marker-layer"
                  />
                  <div
                    class="ace_layer ace_text-layer"
                    style="padding: 0px 4px;"
                  />
                  <div
                    class="ace_layer ace_marker-layer"
                  />
                  <div
                    class="ace_layer ace_cursor-layer ace_hidden-cursors"
                  >
                    <div
                      class="ace_cursor"
                    />
                  </div>
                </div>
              </div>
              <div
                class="ace_scrollbar ace_scrollbar-v"
                style="display: none; width: 20px;"
              >
                <div
                  class="ace_scrollbar-inner"
                  style="width: 20px;"
                />
              </div>
              <div
                class="ace_scrollbar ace_scrollbar-h"
                style="display: none; height: 20px;"
              >
                <div
                  class="ace_scrollbar-inner"
                  style="height: 20px;"
                />
              </div>
              <div
                style="height: auto; width: auto; top: 0px; left: 0px; visibility: hidden; position: absolute; white-space: pre; overflow: hidden;"
              >
                <div
                  style="height: auto; width: auto; top: 0px; left: 0px; visibility: hidden; position: absolute; white-space: pre; overflow: visible;"
                />
                <div
                  style="height: auto; width: auto; top: 0px; left: 0px; visibility: hidden; position: absolute; white-space: pre; overflow: visible;"
                >
                  XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
                </div>
              </div>
            </div>
          </div>
          <div
            class="row"
          >
            <span
              class="key"
            >
              key6
            </span>
            <span
              class="valueText"
            >
              value6
            </span>
          </div>
          <div
            class="row"
          >
            <span
              class="key"
            >
              key6
            </span>
            <span
              class="valueText"
            >
              value7
            </span>
          </div>
          <div
            class="row"
          >
            <span
              class="key"
            >
              key
            </span>
            <span
              class="valueText"
            >
              [object Object]
            </span>
          </div>
        </div>
      </div>
    `);
  });

  it('does render values with the provided valueComponent', () => {
    const ValueComponent: React.FC<any> = ({ value, ...rest }) => (
      <a data-testid='value-component' {...rest}>
        {JSON.stringify(value)}
      </a>
    );
    const { getByTestId } = render(
      <DetailsTable
        fields={[['key2', { key: 'foobar', bucket: 'bucket', endpoint: 's3.amazonaws.com' }]]}
        valueComponent={ValueComponent}
        valueComponentProps={{ extraprop: 'extra' }}
      />,
    );
    expect(getByTestId('value-component')).toMatchInlineSnapshot(`
      <a
        data-testid="value-component"
        extraprop="extra"
      >
        {"key":"foobar","bucket":"bucket","endpoint":"s3.amazonaws.com"}
      </a>
    `);
  });
});
