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

export interface StaticCanvas {
  elements: Elements;
  namespaces: string[];
  doubleClickNode: (node: Node) => void;
  setNamespaces: (namespaces: string[]) => void;
}

const StaticCanvas = ({ elements, namespaces, doubleClickNode, setNamespaces }: StaticCanvas) => {
  return (
    <>
      <SubDagNamespace namespaces={namespaces} setNamespaces={setNamespaces}></SubDagNamespace>
      <ReactFlowProvider>
        <ReactFlow
          style={{ background: '#f2f2f2' }}
          elements={elements}
          snapToGrid={true}
          nodeTypes={nodeTypes}
          edgeTypes={edgeTypes}
          key='edges'
          onElementClick={(event, element) => {
            console.log(event);
            console.log(element);
          }}
          onNodeDoubleClick={(event, element) => {
            doubleClickNode(element as Node);
          }}
        >
          <MiniMap />
          <Controls />
          <Background variant={BackgroundVariant.Dots} />
        </ReactFlow>
      </ReactFlowProvider>
    </>
  );
};
export default StaticCanvas;
