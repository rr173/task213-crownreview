# 城市树冠激光点云断枝复核台 (task213-crownreview)

面向城市林业人员的树冠断枝复核服务：上传按树木编号切分的点云块，服务构造枝干骨架、检测连通性断裂与遮挡不确定区；人员在三维简化视图中确认、合并或否决断枝候选，并发布不可变巡检版本。

## 业务域

激光雷达（LiDAR）点云处理 + 计算几何：点云解析、树冠骨架构图、连通性断裂检测、遮挡不确定区、专家复核与版本冻结。核心实体为点云块 / 骨架边 / 断枝候选 / 巡检版本，均为空间几何数据实体，服务为领域计算工具而非 OA / 业务系统 / 数据可视化前端。

## 标准命令

```bash
# 构建
CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...

# 静态检查
CGO_ENABLED=0 GOTOOLCHAIN=local go vet ./...

# 测试
CGO_ENABLED=0 GOTOOLCHAIN=local go test ./...

# 自检测试（真实建库→上传→解析→骨架→检测→复核→发布→关闭重开验证恢复）
CGO_ENABLED=0 GOTOOLCHAIN=local go run ./cmd/crownreview --smoke-test

# 长驻服务
CGO_ENABLED=0 GOTOOLCHAIN=local go run ./cmd/crownreview --addr :8080 --db ./crownreview.db
```

## API 入口（统一前缀 /api）

- 批次：`POST /api/batches`、`GET /api/batches`、`GET /api/batches/:id`、`POST /api/batches/:id/upload`、`POST /api/batches/:id/publish`
- 点云块：`GET /api/blocks`、`GET /api/blocks/:id`、`POST /api/blocks/:id/parse`、`GET /api/blocks/:id/skeleton`、`GET /api/blocks/:id/occlusion`
- 断枝候选：`GET /api/candidates`、`GET /api/candidates/:id`、`POST /api/candidates/:id/confirm`、`POST /api/candidates/:id/reject`、`POST /api/candidates/:id/merge`
- 复核意见：`POST /api/candidates/:id/opinions`、`GET /api/candidates/:id/opinions`
- 巡检版本：`POST /api/batches/:id/versions`、`GET /api/batches/:id/versions`、`POST /api/versions/:id/freeze`、`POST /api/versions/:id/share`、`POST /api/versions/:id/supersede`
- 自检：`GET /api/health`、`GET /api/stats`

## 持久化

SQLite（纯 Go 驱动 modernc.org/sqlite，无 CGO 依赖），保存点云摘要、骨架边、候选证据、人工意见与版本。相同块哈希幂等；冻结版本保留原始摘要，不可覆盖。详见 `项目设计文档.md`。
