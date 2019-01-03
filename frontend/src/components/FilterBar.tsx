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
import * as Autosuggest from 'react-autosuggest';
import ChipInput, { ChipRendererArgs } from 'material-ui-chip-input';
import Chip from '@material-ui/core/Chip';
import Paper from '@material-ui/core/Paper';
import { SuggestionsFetchRequestedParams, SuggestionHighlightedParams, SuggestionSelectedEventData } from 'react-autosuggest';
import { classes } from 'typestyle';
import MenuItem from '@material-ui/core/MenuItem';
import { ApiFilter } from 'src/apis/filter';
// TODO match and parse required editing the node modules to import correctly.
// Perhaps we should consider using allowSyntheticDefaultImports, although we could pretty easily
// implement these functions ourselves if need be.
// tslint:disable-next-line:no-var-requires
const match = require('autosuggest-highlight/match');
// tslint:disable-next-line:no-var-requires
const parse = require('autosuggest-highlight/parse');

interface FilterChip {
  key: string;
  value: {
    filterValue: number | string;
    type: string;
  };
}

interface FilterBarProps {
  filterTypes: string[];
  // TODO: filter should have type ApiFilter
  filter: (filter: ApiFilter) => void;
}

interface FilterBarState {
  filterChips: FilterChip[];
  lastSelectedFilterType: string;
  shouldSuggestTypes: boolean;
  suggestions: string[];
  textFieldInput: string;
}

export default class FilterBar extends React.Component<FilterBarProps, FilterBarState> {
  private _isMounted = true;

  // TODO: these should not be hardcoded.
  private FILTER_VALUE_SUGGESTIONS = ['My Pipeline', 'Pipeline 1', 'Pipeline 2'];

  constructor(props: FilterBarProps) {
    super(props);

    this.state = {
      filterChips: [],
      lastSelectedFilterType: '',
      // This will control whether we are suggesting the filter types (name, timestamp, etc.) or
      // actual values
      shouldSuggestTypes: true,
      suggestions: props.filterTypes,
      textFieldInput: '',
    };
  }

  public componentWillUnmount(): void {
    this._isMounted = false;
  }

  public render(): JSX.Element {
    const { filterChips, suggestions, textFieldInput } = this.state;
    return (
      <React.Fragment>
        <Autosuggest
          getSuggestionValue={(suggestion) => suggestion}
          inputProps={{
            cancelBubble: true,
            chips: filterChips,
            classes,
            onAdd: this._handleAddChip.bind(this),
            onChange: this._handleTextFieldInputChange.bind(this),
            onDelete: this._handleDeleteChip.bind(this),
            value: textFieldInput,
          }}
          onSuggestionHighlighted={this._handleSuggestionHighlighted.bind(this)}
          onSuggestionSelected={this._onSuggestionSelected.bind(this)}
          onSuggestionsClearRequested={this._handleSuggestionsClearRequested.bind(this)}
          onSuggestionsFetchRequested={this._handleSuggestionsFetchRequested.bind(this)}
          renderInputComponent={this._renderChipInput.bind(this)}
          renderSuggestion={this._renderSuggestion.bind(this)}
          renderSuggestionsContainer={this._renderSuggestionsContainer.bind(this)}
          shouldRenderSuggestions={this._shouldRenderSuggestion.bind(this)}
          suggestions={suggestions}
        />
      </React.Fragment>
    );
  }

  private _renderChipInput(inputProps: any): JSX.Element {
    // TODO we need to pull 'classes' out of the props, but it conflicts with the 'classes' from type style
    // tslint:disable-next-line:no-shadowed-variable
    const { classes, value, cancelBubble, onChange, onAdd, onDelete, ref, ...other } = inputProps;

    return (
    <ChipInput
      classes={{}}
      // value here is the key to updating what is displayed in the bar.
      InputProps={{ value }}
      value={this.state.filterChips}
      onUpdateInput={onChange}
      onAdd={onAdd}
      onDelete={onDelete}
      // this ref appears to be an empty function, but seems to be necessary
      inputRef={ref}
      {...other}
      chipRenderer={(args: ChipRendererArgs, key: any) => {
        return (
        <Chip
          key={key}
          style={{
            backgroundColor: args.isFocused ? 'darkgray' : 'lightgray',
            pointerEvents: args.isDisabled ? 'none' : undefined,
          }}
          onClick={args.handleClick}
          onDelete={args.handleDelete}
          label={(
            <div>
              {/* value here is declared as type string, but is in fact a FilterChip */}
              {(args.value as any).value.type &&
                <span style={{ color: 'gray' }}>{(args.value as any).value.type} : </span>
              }
              <span>{(args.value as any).value.filterValue}</span>
            </div>
          )}
        />
      );}}
    />);
  }

  private _renderSuggestion (suggestion: string, obj: { query: string, isHighlighted: boolean }): JSX.Element {
    // TODO: this matching is fuzzy, but the rest of the code in this file uses 'startsWith'
    const matches = match(suggestion, obj.query);
    const parts = parse(suggestion, matches);
    // tslint:disable-next-line:no-console
    console.log(suggestion, obj.query, matches, parts);

    return (
      <MenuItem
        selected={obj.isHighlighted}
        component='div'
        onMouseDown={(e) => e.preventDefault()} // prevent the click causing the input to be blurred
      >
        <div>
          {parts.map((part: any, index: any) => {
            return part.highlight ? (
              <span key={String(index)} style={{ fontWeight: 600 }}>
                {part.text}
              </span>
            ) : (
              <strong key={String(index)} style={{ fontWeight: 300 }}>
                {part.text}
              </strong>
            );
          })}
        </div>
      </MenuItem>
    );
  }

  private async _handleSuggestionsFetchRequested(param: SuggestionsFetchRequestedParams): Promise<void> {
    // TODO: actually fetch suggestions
    const possibleSuggestions =
      this.state.shouldSuggestTypes
      ? this.props.filterTypes
      : this.FILTER_VALUE_SUGGESTIONS;
    this.setStateSafe({
      suggestions: possibleSuggestions.filter(s => {
        const trimmedInput = param.value.substring(this.state.lastSelectedFilterType.length);
        return s.toLocaleLowerCase().startsWith(trimmedInput.toLocaleLowerCase());
      })
    });
  }

  private async _handleSuggestionsClearRequested(): Promise<void> {
    // TODO: actually reset suggestions
    const possibleSuggestions =
      this.state.shouldSuggestTypes
        ? this.props.filterTypes
        : this.FILTER_VALUE_SUGGESTIONS;
    this.setStateSafe({ suggestions: possibleSuggestions });
  }

  private _renderSuggestionsContainer (options: any): JSX.Element {
    const { containerProps, children } = options;
    return (
      <Paper {...containerProps} square={true} style={{ width: 120 }} >
        {children}
      </Paper>
    );
  }

  private _handleSuggestionHighlighted(params: SuggestionHighlightedParams): void {
    // TODO: this will need to be updated if suggestion becomes more than a string.
    // tslint:disable-next-line:no-console
    console.log('handleSuggestionHighlighted', this.state.shouldSuggestTypes, params);
    if (params && params.suggestion) {
      this.setStateSafe({
        textFieldInput: this.state.shouldSuggestTypes
          ? params.suggestion
          : this.state.lastSelectedFilterType + params.suggestion
      });
    }
  }

  private _handleTextFieldInputChange = (event: React.FormEvent<any>, param: Autosuggest.ChangeEvent) => {
    // tslint:disable-next-line:no-console
    console.log('handletextFieldInputChange', event, param);
    const { lastSelectedFilterType } = this.state;
    if (param.method === 'type') {
      const newValLower = param.newValue.toLocaleLowerCase();
      // Check if the current text in the input field matches any of our filter types, and ensure
      // there is a colon before moving on to filter value suggestions
      const inputFilterTypePrefix =
        this.props.filterTypes.find((s) => newValLower.startsWith(s.toLocaleLowerCase() + ':'));

      this.setStateSafe({
        // lastSelectedFilterType should be cleared if input no longer starts with it.
        lastSelectedFilterType: inputFilterTypePrefix ? inputFilterTypePrefix + ':' : '',
        // Should show types if the input does not start with a filter type.
        shouldSuggestTypes: !inputFilterTypePrefix,
        textFieldInput: param.newValue,
      }, async () => this._handleSuggestionsFetchRequested({ value: param.newValue, reason: 'input-changed' }) );
    } else if (param.method === 'up' || param.method === 'down') {
      this.setStateSafe({
        textFieldInput: lastSelectedFilterType + param.newValue
      });
    }
  };

  private _shouldRenderSuggestion(value: string): boolean {
    const { lastSelectedFilterType, suggestions } = this.state;
    // tslint:disable-next-line:no-console
    console.log('shouldRenderSuggestions', lastSelectedFilterType, value, suggestions);
    return !value
      || !!suggestions.find(s => {
        const trimmedInput = value.substring(lastSelectedFilterType.length);
        return s.toLocaleLowerCase().startsWith(trimmedInput.toLocaleLowerCase());
      });
  }

  private _createFilter(): ApiFilter {
    // TODO: implement
    return {};
  }

  private _handleAddChip(chip: string): void {
    // tslint:disable-next-line:no-console
    console.log('_handleAddChip', chip);
    const { filterChips, lastSelectedFilterType } = this.state;
    filterChips.push({
      key: filterChips.length + '',
      value: {
        filterValue: `${chip}`,
        // Trim off the ":" at the end of the filter type
        type: lastSelectedFilterType.slice(0, -1),
      }
    });
    this.setStateSafe({
        filterChips,
        // Clear the last selected suggestion so the autosuggest will no longer look for it.
        lastSelectedFilterType: '',
        shouldSuggestTypes: true,
        textFieldInput: ''
      },
      async () => this.props.filter(this._createFilter())
    );
  }

  private _handleDeleteChip(chip: FilterChip, index: number): void {
    const chips = this.state.filterChips;
    chips.splice(index, 1);
    this.setStateSafe({ filterChips: chips }, async () => this.props.filter(this._createFilter()));
  }

  private _onSuggestionSelected(event: React.FormEvent<any>, data: SuggestionSelectedEventData<string>): void {
    const { shouldSuggestTypes } = this.state;
    let { lastSelectedFilterType } = this.state;
    let textFieldInput;

    // Don't add a chip if we only selected a type.
    if (shouldSuggestTypes) {
      // Hold onto the last type that was selected so it can be subtracted from the query, since it
      // will display as text in the input box.
      lastSelectedFilterType = data.suggestion + ':';
      textFieldInput = lastSelectedFilterType;
      this.setStateSafe({
        lastSelectedFilterType,
        shouldSuggestTypes: !shouldSuggestTypes,
        textFieldInput,
      });
    } else {
      // _handleAddChip covers clearing the text field and such so we don't do it here.
      this._handleAddChip(data.suggestion);
    }
    return event.preventDefault();
  }

  private setStateSafe(newState: Partial<FilterBarState>, cb?: () => void): void {
    if (this._isMounted) {
      this.setState(newState as any, cb);
    }
  }
}
