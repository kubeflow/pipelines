# Cache Server 手动测试指南

## 快速开始

### 前提条件

1. PostgreSQL 在 Kind 集群中运行
2. 需要的工具：`curl`, `jq`, `psql`

### 步骤 1: 启动本地服务

#### 1.1 Port-forward PostgreSQL

```bash
kubectl port-forward -n kubeflow svc/postgres-service 5432:5432 --address=127.0.0.1 &
```

验证连接：
```bash
PGPASSWORD=password psql -h 127.0.0.1 -U user -d cachedb -c "\dt"
```

应该看到 `execution_caches` 表。

#### 1.2 启动 Cache Server (VSCode)

1. 打开 VSCode
2. 进入 Debug 面板 (Cmd+Shift+D / Ctrl+Shift+D)
3. 选择 **"PG-Launch Cache Server"**
4. 按 F5 启动

**验证启动成功**：Debug Console 应该显示：
```
2025/10/18 XX:XX:XX Initing client manager....
2025/10/18 XX:XX:XX Database created
2025/10/18 XX:XX:XX Table: execution_caches
2025/10/18 XX:XX:XX Using TLS cert directory from environment: ...
2025/10/18 XX:XX:XX Starting cache server on https://0.0.0.0:8443/mutate
2025/10/18 XX:XX:XX Watching namespace: kubeflow
2025/10/18 XX:XX:XX ✅ Cache server is ready to accept requests
```

---

### 步骤 2: 运行自动化测试脚本

```bash
cd backend/src/cache/scripts
chmod +x manual-webhook-test.sh
./manual-webhook-test.sh
```

**脚本会自动测试**:
- ✅ HTTP 方法验证
- ✅ Content-Type 验证
- ✅ JSON 解析
- ✅ Cache 未命中（添加 cache key）
- ✅ Cache 命中（替换为 busybox）
- ✅ TFX Pod 跳过
- ✅ V2 Pod 跳过
- ✅ 数据库状态

**预期输出示例**:
```
🧪 Cache Server Manual Webhook Testing
========================================

0️⃣  Checking if cache server is accessible...
   ✅ Cache server is running

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
TEST 1: GET Request (Should be rejected)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
✅ PASS: Server correctly rejected GET request
   Response: Invalid method "GET", only POST requests are allowed

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
TEST 2: Invalid Content-Type
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
✅ PASS: Server correctly rejected invalid content type
...
```

---

### 步骤 3: 手动单个测试（可选）

如果你想逐个手动测试：

#### 测试 1: 基础连通性

```bash
curl -k https://localhost:8443/mutate
```

预期：`Invalid method "GET", only POST requests are allowed`

#### 测试 2: Cache 未命中

```bash
# 清空数据库
PGPASSWORD=password psql -h 127.0.0.1 -U user -d cachedb \
  -c "TRUNCATE TABLE execution_caches RESTART IDENTITY;"

# 发送请求
cd backend/src/cache/test-data
curl -k -X POST https://localhost:8443/mutate \
  -H "Content-Type: application/json" \
  -d @test-admission-no-cache.json | \
  jq -r '.response.patch' | base64 -d | jq .
```

**验证点**:
- 应该看到 `execution_cache_key` annotation 被添加
- `cache_id` label 为空字符串

#### 测试 3: Cache 命中

```bash
# 插入缓存条目
PGPASSWORD=password psql -h 127.0.0.1 -U user -d cachedb <<EOF
INSERT INTO execution_caches
  (ExecutionCacheKey, ExecutionTemplate, ExecutionOutput, MaxCacheStaleness, StartedAtInSec, EndedAtInSec)
VALUES (
  '1933d178a14bc415466cfd1b3ca2100af975e8c59e1ff9d502fcf18eb5cbd7f7',
  '{"container":{"command":["echo","Hello"],"image":"python:3.9"}}',
  '{"workflows.argoproj.io/outputs":"{\"exitCode\":\"0\"}","pipelines.kubeflow.org/metadata_execution_id":"123"}',
  -1,
  EXTRACT(EPOCH FROM NOW())::bigint,
  EXTRACT(EPOCH FROM NOW())::bigint
);
EOF

# 再次发送请求
curl -k -X POST https://localhost:8443/mutate \
  -H "Content-Type: application/json" \
  -d @test-admission-no-cache.json | \
  jq -r '.response.patch' | base64 -d | jq .
```

**验证点**:
- 容器应该被替换为 `ghcr.io/containerd/busybox`
- `cache_id` label 应该是 "1"
- `reused_from_cache` label 应该是 "true"
- 应该有 `workflows.argoproj.io/outputs` annotation

#### 测试 4: TFX Pod (跳过)

```bash
curl -k -X POST https://localhost:8443/mutate \
  -H "Content-Type: application/json" \
  -d @test-admission-tfx.json | jq .
```

**验证点**: 响应中应该没有 `patch` 字段

#### 测试 5: V2 Pod (跳过)

```bash
curl -k -X POST https://localhost:8443/mutate \
  -H "Content-Type: application/json" \
  -d @test-admission-v2.json | jq .
```

**验证点**: 响应中应该没有 `patch` 字段

---

### 步骤 4: 观察 Cache Server 日志

在 VSCode Debug Console 中，你应该看到：

**Cache 未命中时**:
```
2025/10/18 XX:XX:XX 1933d178a14bc415466cfd1b3ca2100af975e8c59e1ff9d502fcf18eb5cbd7f7
2025/10/18 XX:XX:XX cacheStalenessInSeconds: -1
2025/10/18 XX:XX:XX Number of deleted rows: 0
2025/10/18 XX:XX:XX Execution cache not found with cache key: "1933d178..."
```

**Cache 命中时**:
```
2025/10/18 XX:XX:XX 1933d178a14bc415466cfd1b3ca2100af975e8c59e1ff9d502fcf18eb5cbd7f7
2025/10/18 XX:XX:XX cacheStalenessInSeconds: -1
2025/10/18 XX:XX:XX Number of deleted rows: 0
2025/10/18 XX:XX:XX Get id: 1
2025/10/18 XX:XX:XX Get template: {"container":{"command":["echo","Hello"],"image":"python:3.9"}}
2025/10/18 XX:XX:XX Cached output: {"workflows.argoproj.io/outputs":...}
```

**TFX/V2 Pod 跳过时**:
```
2025/10/18 XX:XX:XX This pod tfx-pod is created by tfx pipelines.
2025/10/18 XX:XX:XX This pod v2-pod is created by KFP v2 pipelines.
```

---

### 步骤 5: 验证数据库状态

```bash
# 查看所有缓存条目
PGPASSWORD=password psql -h 127.0.0.1 -U user -d cachedb \
  -c "SELECT ID, ExecutionCacheKey, MaxCacheStaleness, StartedAtInSec FROM execution_caches;"

# 查看特定条目
PGPASSWORD=password psql -h 127.0.0.1 -U user -d cachedb \
  -c "SELECT * FROM execution_caches WHERE ID = 1;"

# 查看缓存输出
PGPASSWORD=password psql -h 127.0.0.1 -U user -d cachedb \
  -c "SELECT ExecutionOutput FROM execution_caches WHERE ID = 1;"
```

---

## 测试总结检查清单

完成测试后，确认以下项目：

- [ ] ✅ Cache server 启动成功
- [ ] ✅ 可以连接到 PostgreSQL
- [ ] ✅ GET 请求被正确拒绝
- [ ] ✅ 无效 Content-Type 被拒绝
- [ ] ✅ Cache 未命中时添加 cache key
- [ ] ✅ Cache 命中时替换容器为 busybox
- [ ] ✅ Cache 命中时设置正确的 labels
- [ ] ✅ TFX Pods 被跳过
- [ ] ✅ V2 Pods 被跳过
- [ ] ✅ 数据库中正确保存缓存条目

---

## 清理

```bash
# 清空缓存表
PGPASSWORD=password psql -h 127.0.0.1 -U user -d cachedb \
  -c "TRUNCATE TABLE execution_caches RESTART IDENTITY;"

# 停止 port-forward
pkill -f "port-forward.*postgres"

# 停止 cache-server (在 VSCode 中点击停止调试)
```

---

## 故障排查

### 问题: curl 提示 "Connection refused"

**原因**: Cache server 未启动或监听错误端口

**解决**:
1. 检查 VSCode Debug Console 是否有错误
2. 确认看到 "Cache server is ready to accept requests" 消息
3. 检查端口: `lsof -i :8443`

### 问题: "Database created" 未出现在日志中

**原因**: PostgreSQL 连接失败

**解决**:
1. 检查 port-forward: `lsof -i :5432`
2. 测试连接: `PGPASSWORD=password psql -h 127.0.0.1 -U user -d postgres -c "SELECT 1;"`
3. 重启 port-forward

### 问题: Cache 未命中（即使数据库有条目）

**原因**: Cache key 不匹配

**解决**:
1. 检查日志中的 cache key
2. 查询数据库: `SELECT ExecutionCacheKey FROM execution_caches;`
3. 确保两者一致

### 问题: jq 命令找不到

**安装**:
```bash
brew install jq  # macOS
```

---

## 下一步

完成手动测试后，继续：

1. ✅ 构建本地 cache-server Docker 镜像
2. ✅ 部署到 Kind 集群
3. ✅ 运行完整集成测试
4. ✅ 验证 CI workflow

参考: `CACHE_WORKFLOW.md` 了解完整工作流程
