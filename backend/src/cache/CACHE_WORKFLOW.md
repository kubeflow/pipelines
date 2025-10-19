# KFP Cache 工作流程详解

## 概述

KFP Cache Server 通过 Kubernetes Admission Webhook 机制实现 Pipeline 执行结果的缓存和重用。

## 核心组件

### 1. Mutation Webhook (`mutation.go`)
- **触发时机**: Pod 创建时
- **职责**: 检查是否有可用缓存，如果有则修改 Pod 配置

### 2. Watcher (`watcher.go`)
- **触发时机**: Pod 完成后
- **职责**: 将成功的执行结果保存到数据库

### 3. Database Store (`storage/execution_cache_store.go`)
- **职责**: 缓存的存储、查询和管理

---

## 完整工作流程

```
┌─────────────────────────────────────────────────────────────────┐
│                    Pipeline Run 开始                              │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│  1. Argo 创建 Pod (带有 cache_enabled=true 标签)                  │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│  2. Kubernetes API Server 调用 MutatingWebhook                    │
│     - URL: https://cache-server.kubeflow.svc:443/mutate          │
│     - Method: POST                                               │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│  3. Cache Server 处理 Webhook (MutatePodIfCached)                │
│     ├─ 检查 Pod 是否启用缓存                                       │
│     ├─ 过滤 TFX 和 V2 Pod (它们有自己的缓存机制)                   │
│     ├─ 从 Pod annotations 提取 Argo template                     │
│     └─ 生成 cache key (SHA256 hash)                              │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
         ┌───────────────┴───────────────┐
         │   查询数据库是否有缓存           │
         └───────┬───────────────┬────────┘
                 │ 有缓存          │ 无缓存
                 ▼               ▼
    ┌─────────────────┐   ┌──────────────────┐
    │ 4a. 缓存命中      │   │ 4b. 缓存未命中    │
    └────────┬─────────┘   └────────┬─────────┘
             │                      │
             ▼                      ▼
┌──────────────────────┐   ┌────────────────────┐
│ 5a. 修改 Pod:         │   │ 5b. 添加注解:       │
│ • 替换为 busybox 镜像 │   │ • execution_key    │
│ • 注入缓存输出         │   │ • cache_id = ""    │
│ • 添加缓存标签         │   │                    │
│ • 移除 initContainers │   │ Pod 正常执行        │
└──────────┬───────────┘   └────────┬───────────┘
           │                        │
           ▼                        ▼
┌─────────────────────────────────────────────────────────────────┐
│  6. Kubernetes 调度并运行 Pod                                     │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│  7. Watcher 监控 Pod 完成事件                                     │
│     - Label Selector: pipelines.kubeflow.org/cache_id           │
│     - 只处理 Succeeded 且 completed=true 的 Pod                   │
└────────────────────────┬────────────────────────────────────────┘
                         │
         ┌───────────────┴───────────────┐
         │ cache_id 是否为空？             │
         └───────┬───────────────┬────────┘
                 │ 空 (新执行)     │ 非空 (已缓存)
                 ▼               ▼
    ┌─────────────────────┐   ┌──────────────┐
    │ 8a. 保存到数据库     │   │ 8b. 跳过      │
    │ • execution_key     │   │ (已经有缓存)   │
    │ • execution_output  │   │              │
    │ • execution_template│   │              │
    │ • max_cache_staleness│  │              │
    └─────────┬───────────┘   └──────────────┘
              │
              ▼
    ┌─────────────────────┐
    │ 9. Patch Pod 的      │
    │    cache_id label   │
    └─────────────────────┘
```

---

## 关键数据结构

### Pod Labels

```yaml
labels:
  # 标记 Pod 启用缓存
  pipelines.kubeflow.org/cache_enabled: "true"

  # 缓存 ID（空表示未命中，数字表示缓存条目 ID）
  pipelines.kubeflow.org/cache_id: "123"

  # 标记 Pod 使用了缓存
  pipelines.kubeflow.org/reused_from_cache: "true"

  # Argo workflow 相关
  workflows.argoproj.io/completed: "true"
  workflows.argoproj.io/node-name: "pipeline-step-1"
```

### Pod Annotations

```yaml
annotations:
  # 执行缓存键 (SHA256 hash)
  pipelines.kubeflow.org/execution_cache_key: "abc123..."

  # Argo workflow 输出
  workflows.argoproj.io/outputs: '{"artifacts":[...]}'

  # Argo workflow 模板
  workflows.argoproj.io/template: '{"container":{...}}'

  # 最大缓存过期时间
  pipelines.kubeflow.org/max_cache_staleness: "P7D"
```

### Database Schema (execution_caches 表)

```sql
CREATE TABLE execution_caches (
    id SERIAL PRIMARY KEY,
    execution_cache_key VARCHAR(255) NOT NULL,  -- SHA256 hash
    execution_template TEXT NOT NULL,           -- JSON: Argo template
    execution_output TEXT NOT NULL,             -- JSON: outputs + metadata_execution_id
    max_cache_staleness BIGINT DEFAULT -1,      -- 秒数，-1 表示永不过期
    created_at TIMESTAMP DEFAULT NOW(),
    INDEX (execution_cache_key)
);
```

---

## Cache Key 生成逻辑

Cache key 基于以下 Pod template 字段计算 SHA256：

```json
{
  "container": {
    "image": "...",      // 容器镜像
    "command": [...],    // 命令
    "args": [...],       // 参数
    "env": [...],        // 环境变量
    "volumeMounts": [...] // Volume 挂载
  },
  "inputs": {...},       // 输入参数
  "outputs": {...},      // 输出定义
  "volumes": [...],      // Volumes
  "initContainers": [...],
  "sidecars": [...]
}
```

**重要**: 以下内容**不影响** cache key：
- Pod 名称
- 时间戳
- 资源限制 (CPU/Memory)
- Node selector
- Argo 的内部状态

---

## 缓存过期策略

### 1. 用户指定过期时间
通过 annotation 指定：
```yaml
pipelines.kubeflow.org/max_cache_staleness: "P7D"  # 7 天
```

### 2. 默认过期时间
通过环境变量配置：
```bash
DEFAULT_CACHE_STALENESS="P30D"  # 30 天
```

### 3. 最大过期时间
管理员强制的上限：
```bash
MAXIMUM_CACHE_STALENESS="P90D"  # 90 天
```

### 优先级
```
最终过期时间 = min(用户指定 || 默认值, 最大值)
```

---

## 特殊情况处理

### 1. TFX Pods (跳过)
- 检测方式: `pipelines.kubeflow.org/pipeline-sdk-type` 包含 "tfx"
- 原因: TFX 有自己的缓存机制

### 2. KFP V2 Pods (跳过)
- 检测方式: `pipelines.kubeflow.org/v2_component: "true"`
- 原因: V2 由 driver 处理缓存

### 3. PVC 挂载
- PVC 名称包含在 `volumeMounts` 中，影响 cache key
- 相同 PVC 名称 → 缓存命中
- 不同 PVC 名称 → 缓存未命中

---

## 测试验证点

### 单元测试 (`*_test.go`)
- ✅ Cache key 生成是否一致
- ✅ 数据库 CRUD 操作
- ✅ Webhook mutation 逻辑

### 集成测试 (`cache_test.go`)
- ✅ 第二次运行使用缓存
- ✅ MLMD 中状态为 `CACHED`
- ✅ PVC 名称影响缓存

### 手动验证
- ✅ Pod 被替换为 busybox
- ✅ 数据库中有缓存条目
- ✅ Watcher 正确保存输出

---

## 常见问题

### Q1: 为什么第二次运行没用缓存？
**检查**:
1. 第一次运行是否成功（`Pod.Status.Phase == Succeeded`）
2. Pod 是否有 `cache_enabled=true` 标签
3. Cache key 是否一致（查看 cache-server 日志）
4. 缓存是否过期

### Q2: Watcher 为什么没保存缓存？
**检查**:
1. Pod 是否有 `workflows.argoproj.io/completed=true` 标签
2. Pod 是否在监控的 namespace
3. Watcher 是否运行（`WatchPods` goroutine）

### Q3: MutatingWebhook 为什么没被调用？
**检查**:
1. MutatingWebhookConfiguration 是否存在
2. Webhook service 是否可达
3. TLS 证书是否有效
4. Namespace 是否在 webhook 配置的范围内

---

## 相关代码文件

- `main.go` - 入口，启动 webhook server 和 watcher
- `server/mutation.go` - Webhook 处理逻辑
- `server/watcher.go` - Pod 监控和缓存保存
- `server/admission.go` - Webhook admission 框架
- `storage/execution_cache_store.go` - 数据库操作
- `client_manager.go` - 客户端管理（DB + K8s）
- `model/execution_cache.go` - 数据模型
