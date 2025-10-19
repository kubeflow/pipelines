# Cache Server 验证快速开始 ⚡

## 🎯 目标
验证 cache-server 代码逻辑正确，为 CI 通过做准备

---

## 📋 5 分钟快速验证

### 1️⃣ 启动服务（2 分钟）

```bash
# Terminal 1: Port-forward PostgreSQL
kubectl port-forward -n kubeflow svc/postgres-service 5432:5432 &

# Terminal 2: VSCode Debug
# 1. 打开 Debug 面板 (Cmd+Shift+D)
# 2. 选择 "PG-Launch Cache Server"
# 3. 按 F5
```

**✅ 成功标志**: Debug Console 显示
```
✅ Cache server is ready to accept requests
```

### 2️⃣ 运行自动化测试（2 分钟）

```bash
cd backend/src/cache/scripts
chmod +x manual-webhook-test.sh
./manual-webhook-test.sh
```

**✅ 成功标志**: 所有测试显示 `✅ PASS`

### 3️⃣ 验证核心功能（1 分钟）

```bash
# 快速检查缓存命中
cd backend/src/cache/test-data

# Test 1: Cache Miss
curl -k -X POST https://localhost:8443/mutate \
  -H "Content-Type: application/json" \
  -d @test-admission-no-cache.json | jq -r '.response.patch' | base64 -d | jq '.[] | select(.path=="/metadata/labels") | .value."pipelines.kubeflow.org/cache_id"'
# 预期: "" (空字符串)

# Test 2: 插入缓存
PGPASSWORD=password psql -h 127.0.0.1 -U user -d cachedb <<EOF
INSERT INTO execution_caches (ExecutionCacheKey, ExecutionTemplate, ExecutionOutput, MaxCacheStaleness, StartedAtInSec, EndedAtInSec)
VALUES ('1933d178a14bc415466cfd1b3ca2100af975e8c59e1ff9d502fcf18eb5cbd7f7', '{}', '{}', -1, EXTRACT(EPOCH FROM NOW())::bigint, EXTRACT(EPOCH FROM NOW())::bigint);
EOF

# Test 3: Cache Hit
curl -k -X POST https://localhost:8443/mutate \
  -H "Content-Type: application/json" \
  -d @test-admission-no-cache.json | jq -r '.response.patch' | base64 -d | jq '.[] | select(.path=="/metadata/labels") | .value."pipelines.kubeflow.org/cache_id"'
# 预期: "1"
```

---

## 🚀 完整验证流程

按照以下顺序完成验证：

### 阶段 1: 本地验证 ✅ (已完成)
- [x] 单元测试通过 (25个测试)
- [x] 手动 webhook 测试
- [x] 数据库读写验证

### 阶段 2: Kind 集群部署 (下一步)

```bash
# 构建并部署本地镜像
cd backend/src/cache/scripts
chmod +x deploy-local-cache-server.sh
./deploy-local-cache-server.sh
```

验证部署：
```bash
kubectl logs -n kubeflow -l app=cache-server --tail=20 | grep "Cache server is ready"
```

### 阶段 3: 集成测试

```bash
# 启动 port-forward
kubectl port-forward -n kubeflow svc/ml-pipeline 8888:8888 &
kubectl port-forward -n kubeflow svc/metadata-grpc-service 8080:8080 &
kubectl port-forward -n kubeflow svc/postgres-service 5432:5432 --address=127.0.0.3 &

# 运行集成测试
cd backend/test/v2/integration
go test -v -run TestCache -timeout 30m -args \
  -runIntegrationTests=true \
  -namespace=kubeflow \
  -cacheEnabled=true
```

### 阶段 4: CI 验证

Push 代码并观察 CI：
```bash
git add .
git commit -m "fix: cache-server improvements"
git push
```

---

## 📚 完整文档

- **[MANUAL_TESTING.md](./MANUAL_TESTING.md)** - 详细手动测试指南
- **[CACHE_WORKFLOW.md](./CACHE_WORKFLOW.md)** - Cache 工作流程详解
- **[LOCAL_DEV.md](./LOCAL_DEV.md)** - 本地开发环境配置

---

## 🔧 常用命令

### 数据库操作
```bash
# 连接数据库
PGPASSWORD=password psql -h 127.0.0.1 -U user -d cachedb

# 查看缓存
\c cachedb
SELECT * FROM execution_caches;

# 清空缓存
TRUNCATE TABLE execution_caches RESTART IDENTITY;
```

### 调试
```bash
# 查看 cache-server logs
kubectl logs -n kubeflow -l app=cache-server -f

# 查看数据库状态
PGPASSWORD=password psql -h 127.0.0.1 -U user -d cachedb -c "\dt"

# 检查端口
lsof -i :8443  # cache-server
lsof -i :5432  # postgres
```

### 清理
```bash
# 停止 port-forward
pkill -f "port-forward"

# 清理数据库
PGPASSWORD=password psql -h 127.0.0.1 -U user -d cachedb \
  -c "TRUNCATE TABLE execution_caches RESTART IDENTITY;"
```

---

## ✅ 验证检查清单

完成以下检查确保 cache-server 正常工作：

**基础功能**
- [ ] Cache server 启动成功
- [ ] 数据库连接正常
- [ ] Webhook endpoint 响应正常

**核心逻辑**
- [ ] Cache 未命中时添加 execution_cache_key
- [ ] Cache 命中时替换容器为 busybox
- [ ] TFX/V2 Pods 被正确跳过
- [ ] 数据库正确保存和查询缓存

**集成测试**
- [ ] TestCacheSingleRun 通过
- [ ] TestCacheRecurringRun 通过
- [ ] TestCacheSingleRunWithPVC_SameName_Caches 通过

**CI 验证**
- [ ] Cache enabled=true 矩阵通过
- [ ] Cache enabled=false 矩阵通过

---

## 🆘 快速故障排查

| 问题 | 快速检查 | 解决方案 |
|------|---------|---------|
| Cache server 启动失败 | VSCode Debug Console 错误 | 检查 PostgreSQL 连接 |
| Webhook 不响应 | `curl -k https://localhost:8443/mutate` | 重启 cache-server |
| 数据库连接失败 | `psql -h 127.0.0.1 -U user` | 重启 port-forward |
| Cache 未命中 | 查看日志中的 cache key | 验证 template 一致性 |
| 集成测试失败 | 查看 test logs | 检查服务 port-forward |

---

现在开始验证！🎉
