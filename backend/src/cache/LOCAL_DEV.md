# Cache Server 本地开发指南

本指南帮助你在本地运行和调试 cache-server。

## 前置条件

1. **Kind 集群运行中**，并且有 PostgreSQL 部署在 `kubeflow` namespace
2. **Go 开发环境** (已安装)
3. **kubectl** 配置正确

## 快速开始

### 1. 生成 TLS 证书

首次运行需要生成自签名证书（仅用于本地开发）：

```bash
cd backend/src/cache/scripts
./generate-tls-certs.sh
```

证书会生成在 `backend/src/cache/certs/` 目录：
- `cert.pem` - TLS 证书
- `key.pem` - 私钥

### 2. 启动 Port Forward (可选)

如果你不使用 VSCode 的 preLaunchTask，需要手动启动 port-forward：

```bash
kubectl port-forward -n kubeflow svc/postgres-service 5432:5432
```

保持这个终端窗口运行。

### 3. 在 VSCode 中启动

1. 打开 VSCode Debug 面板 (Cmd+Shift+D / Ctrl+Shift+D)
2. 选择 **"PG-Launch Cache Server"** 配置
3. 点击运行 (F5)

VSCode 会自动：
- 启动 port-forward (通过 preLaunchTask)
- 连接到本地 PostgreSQL (127.0.0.1:5432)
- 启动 cache-server 在 https://localhost:8443

## 配置说明

### Launch Configuration

在 `.vscode/launch.json` 中配置：

```json
{
  "name": "PG-Launch Cache Server",
  "env": {
    "TLS_CERT_DIR": "${workspaceFolder}/backend/src/cache/certs"
  },
  "args": [
    "-db_driver=pgx",
    "-db_host=127.0.0.1",
    "-db_port=5432",
    "-db_name=cachedb",
    "-db_user=user",
    "-db_password=password",
    "-namespace_to_watch=kubeflow",
    "-listen_port=8443"
  ]
}
```

### 数据库配置

数据库连接信息来自 kind 集群中的 `postgres-service`：
- **Host**: 127.0.0.1 (通过 port-forward)
- **Port**: 5432
- **Database**: cachedb (自动创建)
- **User**: user
- **Password**: password

### 修改配置

如果你的数据库配置不同，修改 `.vscode/launch.json` 中的相应参数。

## 验证运行

### 检查初始化

cache-server 启动后应该会看到：

```
Initing client manager....
Using TLS cert directory from environment: /path/to/backend/src/cache/certs
Database created
Table: execution_caches
```

### 测试连接

Cache server 监听在 `https://localhost:8443`，主要端点：
- `/mutate` - Webhook mutation endpoint

## 常见问题

### 1. 数据库连接失败

**错误**: `dial tcp 127.0.0.1:5432: connect: connection refused`

**解决方案**:
- 确认 port-forward 正在运行: `kubectl port-forward -n kubeflow svc/postgres-service 5432:5432`
- 检查 PostgreSQL pod 运行正常: `kubectl get pods -n kubeflow | grep postgres`

### 2. 证书文件找不到

**错误**: `open /etc/webhook/certs/cert.pem: no such file or directory`

**解决方案**:
- 运行证书生成脚本: `./backend/src/cache/scripts/generate-tls-certs.sh`
- 确认环境变量设置: `TLS_CERT_DIR` 应该指向 certs 目录

### 3. Kubernetes 客户端连接失败

**错误**: `unable to load in-cluster configuration`

**解决方案**:
- 确保 `~/.kube/config` 存在并配置正确
- 确认 kind 集群正在运行: `kubectl cluster-info`

## 手动运行 (不使用 VSCode)

如果你想在命令行运行：

```bash
# 1. 启动 port-forward
kubectl port-forward -n kubeflow svc/postgres-service 5432:5432 &

# 2. 设置环境变量
export TLS_CERT_DIR="$(pwd)/backend/src/cache/certs"

# 3. 运行 cache-server
cd backend/src/cache
go run . \
  -db_driver=pgx \
  -db_host=127.0.0.1 \
  -db_port=5432 \
  -db_name=cachedb \
  -db_user=user \
  -db_password=password \
  -namespace_to_watch=kubeflow \
  -listen_port=8443 \
  -logtostderr=true
```

## 下一步

- 查看 [README.md](./README.md) 了解如何部署到集群
- 查看代码目录结构了解各个组件
