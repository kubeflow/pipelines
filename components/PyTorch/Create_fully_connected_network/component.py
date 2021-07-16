from kfp.components import create_component_from_func, OutputPath

def create_fully_connected_pytorch_network(
    layer_sizes: list,
    network_path: OutputPath('PyTorchScriptModule'),
    activation_name: str = 'relu',
    random_seed: int = 0,
):
    '''Creates fully-connected network in PyTorch ScriptModule format'''
    import torch
    torch.manual_seed(random_seed)

    activation = getattr(torch, activation_name, None) or getattr(torch.nn.functional, activation_name, None)
    if not activation:
        raise ValueError(f'Activation "{activation_name}" was not found.')

    class ActivationLayer(torch.nn.Module):
        def forward(self, input):
            return activation(input)

    layers = []
    for layer_idx in range(len(layer_sizes) - 1):
        layer = torch.nn.Linear(layer_sizes[layer_idx], layer_sizes[layer_idx + 1])
        layers.append(layer)
        if layer_idx < len(layer_sizes) - 2:
            layers.append(ActivationLayer())

    network = torch.nn.Sequential(*layers)
    script_module = torch.jit.script(network)
    print(script_module)
    script_module.save(network_path)


if __name__ == '__main__':
    create_fully_connected_pytorch_network_op = create_component_from_func(
        create_fully_connected_pytorch_network,
        output_component_file='component.yaml',
        base_image='pytorch/pytorch:1.7.1-cuda11.0-cudnn8-runtime',
        packages_to_install=[],
        annotations={
            "author": "Alexey Volkov <alexey.volkov@ark-kun.com>",
            "canonical_location": "https://raw.githubusercontent.com/Ark-kun/pipeline_components/master/components/PyTorch/Create_fully_connected_network/component.yaml",
        },
    )
