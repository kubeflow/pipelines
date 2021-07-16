from kfp.components import create_component_from_func, InputPath, OutputPath

def train_pytorch_model_from_csv(
    model_path: InputPath('PyTorchScriptModule'),
    training_data_path: InputPath('CSV'),
    trained_model_path: OutputPath('PyTorchScriptModule'),
    label_column_name: str,
    loss_function_name: str = 'mse_loss',
    number_of_epochs: int = 1,
    learning_rate: float = 0.1,
    optimizer_name: str = 'Adadelta',
    optimizer_parameters: dict = None,
    batch_size: int = 32,
    batch_log_interval: int = 100,
    random_seed: int = 0,
):
    '''Trains PyTorch model'''
    import pandas
    import torch

    torch.manual_seed(random_seed)

    use_cuda = torch.cuda.is_available()
    device = torch.device("cuda" if use_cuda else "cpu")

    model = torch.jit.load(model_path)
    model.to(device)
    model.train()

    optimizer_class = getattr(torch.optim, optimizer_name, None)
    if not optimizer_class:
        raise ValueError(f'Optimizer "{optimizer_name}" was not found.')

    optimizer_parameters = optimizer_parameters or {}
    optimizer_parameters['lr'] = learning_rate
    optimizer = optimizer_class(model.parameters(), **optimizer_parameters)

    loss_function = getattr(torch, loss_function_name, None) or getattr(torch.nn, loss_function_name, None) or getattr(torch.nn.functional, loss_function_name, None)
    if not loss_function:
        raise ValueError(f'Loss function "{loss_function_name}" was not found.')

    class CsvDataset(torch.utils.data.Dataset):

        def __init__(self, file_path: str, label_column_name: str, drop_nan_clumns_or_rows: str = 'columns'):
            dataframe = pandas.read_csv(file_path)
            # Preventing error: default_collate: batch must contain tensors, numpy arrays, numbers, dicts or lists; found object
            if drop_nan_clumns_or_rows == 'columns':
                non_nan_data = dataframe.dropna(axis='columns')
                removed_columns = set(dataframe.columns) - set(non_nan_data.columns)
                if removed_columns:
                    print('Skipping columns with NaNs: ' + str(removed_columns))
                dataframe = non_nan_data
            if drop_nan_clumns_or_rows == 'rows':
                non_nan_data = dataframe.dropna(axis='index')
                number_of_removed_rows = len(dataframe) - len(non_nan_data)
                if number_of_removed_rows:
                    print(f'Skipped {number_of_removed_rows} rows with NaNs.')
                dataframe = non_nan_data
            numerical_data = dataframe.select_dtypes(include='number')
            non_numerical_data = dataframe.select_dtypes(exclude='number')
            if not non_numerical_data.empty:
                print('Skipping non-number columns:')
                print(non_numerical_data.dtypes)
            self._dataframe = dataframe
            self.labels = numerical_data[[label_column_name]]
            self.features = numerical_data.drop(columns=[label_column_name])

        def __len__(self):
            return len(self._dataframe)

        def __getitem__(self, index):
            return [self.features.loc[index].to_numpy(dtype='float32'), self.labels.loc[index].to_numpy(dtype='float32')]

    dataset = CsvDataset(
        file_path=training_data_path,
        label_column_name=label_column_name,
    )
    train_loader = torch.utils.data.DataLoader(
        dataset=dataset,
        batch_size=batch_size,
        shuffle=True,
    )

    last_full_batch_loss = None
    for epoch in range(1, number_of_epochs + 1):
        for batch_idx, (data, target) in enumerate(train_loader):
            data, target = data.to(device), target.to(device)
            optimizer.zero_grad()
            output = model(data)
            loss = loss_function(output, target)
            loss.backward()
            optimizer.step()
            if len(data) == batch_size:
                last_full_batch_loss = loss.item()
            if batch_idx % batch_log_interval == 0:
                print('Train Epoch: {} [{}/{} ({:.0f}%)]\tLoss: {:.6f}'.format(
                    epoch, batch_idx * len(data), len(train_loader.dataset),
                    100. * batch_idx / len(train_loader), loss.item()))
        print(f'Training epoch {epoch} completed. Last full batch loss: {last_full_batch_loss:.6f}')

    # print(optimizer.state_dict())
    model.save(trained_model_path)


if __name__ == '__main__':
    train_pytorch_model_from_csv_op = create_component_from_func(
        train_pytorch_model_from_csv,
        output_component_file='component.yaml',
        base_image='pytorch/pytorch:1.7.1-cuda11.0-cudnn8-runtime',
        packages_to_install=['pandas==1.1.5'],
        annotations={
            "author": "Alexey Volkov <alexey.volkov@ark-kun.com>",
            "canonical_location": "https://raw.githubusercontent.com/Ark-kun/pipeline_components/master/components/PyTorch/Train_PyTorch_model/from_CSV/component.yaml",
        },
    )
