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

import { useState } from 'react';
import { Button } from 'src/components/ui/button';
import { Switch } from 'src/components/ui/switch';
import { getFeatureList, saveFeatures } from 'src/features';
import './SecondaryPages.css';

// The application initializes feature storage once before rendering.
export default function FrontendFeatures() {
  const [features, setFeatures] = useState(getFeatureList);
  const reset = () => setFeatures(getFeatureList());
  const submit = () => {
    saveFeatures(features);
    setFeatures(getFeatureList());
  };

  return (
    <section className='kfp-secondary-page' aria-label='Frontend features'>
      <h1>Frontend features</h1>
      <div className='kfp-secondary-actions'>
        <Button onClick={submit}>Save changes</Button>
        <Button variant='secondary' onClick={reset}>
          Reset
        </Button>
      </div>
      <div className='kfp-feature-table-scroll'>
        <table className='kfp-feature-table' aria-label='Frontend features'>
          <thead>
            <tr>
              <th>Feature flag name</th>
              <th>Description</th>
              <th>Enabled</th>
            </tr>
          </thead>
          <tbody>
            {features.map((feature) => (
              <tr key={feature.name}>
                <th scope='row'>{feature.name}</th>
                <td>{feature.description}</td>
                <td>
                  <Switch
                    checked={feature.active}
                    aria-label={`Enable ${feature.name}`}
                    onCheckedChange={(active) =>
                      setFeatures((previous) =>
                        previous.map((item) =>
                          item.name === feature.name ? { ...item, active } : item,
                        ),
                      )
                    }
                  />
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}
