/*
 * Copyright 2023 The Kubeflow Authors
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
import {KubernetesEvent} from "./tabs/RuntimeNodeDetailsV2";
import Banner from "./Banner";
import {commonCss, padding} from "../Css";
import DetailsTable from "./DetailsTable";
import Card from '@material-ui/core/Card';
import CardContent from '@material-ui/core/CardContent';

interface LogViewerProps {
    events: KubernetesEvent[];
}

function sortEvent(a: KubernetesEvent, b: KubernetesEvent): number {
    return new Date(b.creationTimestamp).getTime() - new Date(a.creationTimestamp).getTime();
}

class EventsList extends React.Component<LogViewerProps> {

    public render(): JSX.Element {
        return (
            <div className={commonCss.page}>
                <div className={padding(20)}>
                    {
                        (this.props.events.length === 0) && <Banner message='There is no events for this pod.' mode='info' />
                    }
                    {
                        this.props.events.sort(sortEvent).map((event) => {
                            return (
                                <Card
                                    style={{marginBottom: "1rem"}}
                                    key={`input-artifacts-${event.name}`}>
                                    <CardContent>
                                        <DetailsTable<string>
                                            title={event.message}
                                            fields={Array.of(
                                                ["Source", `${event.source.host} ${event.source.component}`],
                                                ["Count", `${event.count}`],
                                                ["Last seen", `${event.lastTimestamp}`],
                                            )}
                                        />
                                    </CardContent>
                                </Card>
                            )
                        })
                    }
                </div>
            </div>
        );
    }
}

export default EventsList;
