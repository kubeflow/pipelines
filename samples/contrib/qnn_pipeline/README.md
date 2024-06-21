# Qnn-on-kubeflow-with-Node-red

Kube-node-red is aiming to integrate Kubeflow/Kubebeters with node-red, leveraging node-red's low-code modules, and using Kubeflow resources (e.g. Kubeflow pipeline, Kserve) to enhance its AI/ML ability.
## Table of Contents
<!-- toc -->
- [Architecture](#Architecture)
- [Self-defined Node](#Self-defined-Node)
  * [Prerequisites](#Prerequisites)
  * [snippet.js](#snippet.js)
  * [example.js](#example.js)
  * [example.html](#example.html)
- [Reference](#Reference)

<!-- tocstop -->

# Architecture
![image](https://hackmd.io/_uploads/HJeFeO6Up.png)

## Implementation

Use WSL to implement

```
git clone https://github.com/sefgsefg/Qnn-on-kubeflow-with-Node-red.git
```

```
cd Qnn-on-kubeflow-with-Node-red/Qnn/examples
```

```
./run.sh main
```

Problem solve: -bash: ./run.sh: Permission denied
```
chmod +x run.sh
```

```
cd scripts
```


```
chmod +x entrypoint.sh
```

```
cd ..
```
Run ./run.sh main again
```
./run.sh main
```


# Self-defined Node
## Prerequisites
- `snippet.js`
Record the machine learning task code written in Python to modify the task process based on front-end input
- `<example>.js`
Node back-end logic and front-end input processing
- `<example>.html`
Node front-end logic and user configuration logic and user configuration

## snippet.js

1.Machine learning tasks written in python are executed in kubeflow, and js strings are stored in constants for node calls and modifications.

```javascript=
const EXAMPLE =
`
    data = pd.DataFrame(data, columns=col_names[:])
    data.fillna(value=0, inplace=True)
    data = data.values  
`;
```
2. The user-modified part will need to be replaced with the template literal value "%s"
```javascript=
`   data.fillna(value=%s, inplace=True)
`
```
3. Output this constant
```javascript=
module.exports = {
    
    EXAMPLE,
    
};
```        
##  example.js

1. Import the "util" module and import the prepared snippet.js according to the file structure
```javascript=
const util = require('util');
const snippets = require('../snippets');
```

2. Replace the user's front-end configuration with the target code
```javascript=
example = util.format(snippets.EXAMPLE,config.userinput)
```

3. Store the modified code as a variable in the msg object attribute
```javascript=
node.on('input', function(msg) {
            
            msg.payload += example
            node.send(msg);
    
});
```

##  example.html
1. Set relevant configurations according to machine learning task requirements
```javascript=
defaults: {
            userinput: { value: {} }                    
        },
```


2.Write corresponding html template
```html=
<div class="form-row">
        <label for="node-input-userinput>UserInput</label>
        <input type="text" id="node-input-userinput" />
</div>
```
## Implementation

https://github.com/sefgsefg/Qnn-on-kubeflow-with-Node-red/assets/85359362/31677bea-7f14-4d56-b5fa-c0bb1c35352f




## Reference
https://github.com/NightLightTw/kube-nodered

https://github.com/kubeflow/pipelines/tree/1.8.21/backend/api/python_http_client

[Kubeflow implementation：add Random Forest algorithm](https://hackmd.io/@ZJ2023/BJYQGMvJ6)

https://github.com/justin0322/Node-RED-Kubeflow-Pipeline-Extension

