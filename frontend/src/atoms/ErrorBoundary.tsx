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

import React from 'react';
import './ErrorBoundary.css';

interface ErrorBoundaryState {
  error: unknown;
  errorInfo: React.ErrorInfo | null;
}

type ErrorBoundaryProps = React.PropsWithChildren<{ resetKey?: string }>;

export class ErrorBoundary extends React.Component<ErrorBoundaryProps, ErrorBoundaryState> {
  constructor(props: ErrorBoundaryProps) {
    super(props);
    this.state = { error: null, errorInfo: null };
  }

  componentDidUpdate(previousProps: ErrorBoundaryProps) {
    // Navigation recovers a failed page without remounting healthy page state.
    if (this.state.errorInfo && previousProps.resetKey !== this.props.resetKey) {
      this.setState({ error: null, errorInfo: null });
    }
  }

  componentDidCatch(error: unknown, errorInfo: React.ErrorInfo) {
    this.setState({
      error: error,
      errorInfo: errorInfo,
    });
  }

  render() {
    if (this.state.errorInfo) {
      const stack = this.state.errorInfo.componentStack || '';
      const diagnostics = this.state.error ? `${String(this.state.error)}\n${stack}` : stack;
      return (
        <section className='kfp-error-boundary' role='alert'>
          <p>Something went wrong.</p>
          <details>
            <summary>Details</summary>
            <pre>{diagnostics}</pre>
          </details>
        </section>
      );
    }
    return this.props.children;
  }
}
