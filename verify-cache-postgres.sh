#!/bin/bash

# =====================================================
# 验证 Cache-Server PostgreSQL 支持
# =====================================================

set -e

GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}=== Cache-Server PostgreSQL 支持验证 ===${NC}\n"

# Step 1: 检查是否已经部署
echo -e "${YELLOW}[1/5] 检查现有部署状态...${NC}"
if kubectl get namespace kubeflow &> /dev/null; then
    echo -e "${GREEN}✓ kubeflow namespace 存在${NC}"

    # 检查 PostgreSQL
    if kubectl get deployment postgres -n kubeflow &> /dev/null; then
        echo -e "${GREEN}✓ PostgreSQL deployment 已存在${NC}"
        POSTGRES_STATUS=$(kubectl get deployment postgres -n kubeflow -o jsonpath='{.status.conditions[?(@.type=="Available")].status}')
        if [ "$POSTGRES_STATUS" == "True" ]; then
            echo -e "${GREEN}✓ PostgreSQL 运行正常${NC}"
        else
            echo -e "${RED}⚠ PostgreSQL 未就绪，需要重新部署${NC}"
        fi
    else
        echo -e "${YELLOW}PostgreSQL 未部署，需要完整部署${NC}"
    fi

    # 检查 cache-server
    if kubectl get deployment cache-server -n kubeflow &> /dev/null; then
        echo -e "${GREEN}✓ cache-server deployment 已存在${NC}"
    else
        echo -e "${YELLOW}cache-server 未部署${NC}"
    fi
else
    echo -e "${YELLOW}kubeflow namespace 不存在，需要完整部署${NC}"
fi
echo ""

# Step 2: 构建本地镜像
echo -e "${YELLOW}[2/5] 构建 cache-server 镜像...${NC}"
echo -e "${BLUE}运行: make -C backend image_cache${NC}"
if make -C backend image_cache; then
    echo -e "${GREEN}✓ cache-server 镜像构建成功${NC}\n"
else
    echo -e "${RED}✗ 镜像构建失败${NC}"
    exit 1
fi

# Step 3: 加载镜像到 kind cluster
echo -e "${YELLOW}[3/5] 加载镜像到 kind cluster...${NC}"
KIND_CLUSTER="dev-pipelines-api"
if kind get clusters | grep -q "^${KIND_CLUSTER}$"; then
    echo -e "${BLUE}找到 kind cluster: ${KIND_CLUSTER}${NC}"

    # 获取镜像名称
    CACHE_IMAGE=$(docker images --format "{{.Repository}}:{{.Tag}}" | grep cache-server | head -1)
    if [ -z "$CACHE_IMAGE" ]; then
        echo -e "${RED}✗ 找不到 cache-server 镜像${NC}"
        exit 1
    fi
    echo -e "${BLUE}加载镜像: ${CACHE_IMAGE}${NC}"

    # Tag 为 kind registry 格式
    docker tag "$CACHE_IMAGE" "kind-registry:5000/cache-server:latest"
    docker push "kind-registry:5000/cache-server:latest"

    echo -e "${GREEN}✓ 镜像已加载到 kind cluster${NC}\n"
else
    echo -e "${RED}✗ kind cluster '${KIND_CLUSTER}' 不存在${NC}"
    exit 1
fi

# Step 4: 部署 KFP with PostgreSQL
echo -e "${YELLOW}[4/5] 部署 KFP with PostgreSQL...${NC}"
echo -e "${BLUE}运行: ./.github/resources/scripts/deploy-kfp.sh --db-type pgx${NC}"
if ./.github/resources/scripts/deploy-kfp.sh --db-type pgx; then
    echo -e "${GREEN}✓ KFP 部署成功${NC}\n"
else
    echo -e "${RED}✗ KFP 部署失败${NC}"
    exit 1
fi

# Step 5: 验证 cache-server 日志
echo -e "${YELLOW}[5/5] 验证 cache-server 启动和数据库连接...${NC}"
sleep 10  # 等待 pod 启动

echo -e "${BLUE}获取 cache-server pod 名称...${NC}"
CACHE_POD=$(kubectl get pod -n kubeflow -l app=cache-server -o jsonpath='{.items[0].metadata.name}')
if [ -z "$CACHE_POD" ]; then
    echo -e "${RED}✗ 找不到 cache-server pod${NC}"
    exit 1
fi
echo -e "${GREEN}✓ 找到 pod: ${CACHE_POD}${NC}\n"

echo -e "${BLUE}=== Cache-Server 日志（最后 50 行）===${NC}"
kubectl logs -n kubeflow "$CACHE_POD" --tail=50

echo -e "\n${BLUE}检查关键日志...${NC}"
if kubectl logs -n kubeflow "$CACHE_POD" | grep -q "Database created"; then
    echo -e "${GREEN}✓ 数据库创建成功${NC}"
else
    echo -e "${YELLOW}⚠ 未找到 'Database created' 日志${NC}"
fi

if kubectl logs -n kubeflow "$CACHE_POD" | grep -qi "error\|fatal"; then
    echo -e "${RED}⚠ 发现错误日志：${NC}"
    kubectl logs -n kubeflow "$CACHE_POD" | grep -i "error\|fatal"
else
    echo -e "${GREEN}✓ 未发现错误日志${NC}"
fi

# 验证 PostgreSQL 数据库
echo -e "\n${BLUE}验证 PostgreSQL 中的 cachedb 数据库...${NC}"
POSTGRES_POD=$(kubectl get pod -n kubeflow -l app=postgres -o jsonpath='{.items[0].metadata.name}')
if [ -n "$POSTGRES_POD" ]; then
    kubectl exec -n kubeflow "$POSTGRES_POD" -- psql -U user -d postgres -c "\l" | grep cachedb && \
        echo -e "${GREEN}✓ cachedb 数据库存在${NC}" || \
        echo -e "${RED}✗ cachedb 数据库不存在${NC}"
fi

echo -e "\n${GREEN}============================================${NC}"
echo -e "${GREEN}✓ 验证完成！${NC}"
echo -e "${GREEN}============================================${NC}\n"

echo -e "${BLUE}下一步：运行集成测试${NC}"
echo -e "go test -v ./backend/test/v2/integration/... -run TestCache -args -runIntegrationTests=true -namespace=kubeflow -cacheEnabled=true"
echo ""
