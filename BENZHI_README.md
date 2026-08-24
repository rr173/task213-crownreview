基于 Go 实现的城市树冠激光点云断枝复核全栈 Web 项目，一款林业工程分析服务，处理点云骨架构建、断枝候选判定与巡检版本发布。

# 评测说明 (BENZHI_README) — task213-crownreview

## 构建与运行

```bash
# 评测镜像构建（单平台）
bash build_benzhi_docker.sh task213-crownreview linux/amd64

# 双架构验证由出题侧验收脚本执行，项目本身只需提供上述构建脚本和 --smoke-test。
```

## API 契约

路由前缀统一为 `/api`（见 README）。所有写操作返回 JSON，错误含可读 `message` 与 `code`。

## Docker 双架构契约

- `ENTRYPOINT` 为 `/app/crownreview`，`CMD` 默认 `["--smoke-test"]`。
- 双架构验证仅传 `--smoke-test` 标志，不追加 `/app/crownreview` 路径参数（追加会被当作位置参数，导致 flag 不生效、服务长驻后被 kill 137）。
- 平台：`linux/amd64`、`linux/arm64` 各执行构建 + `docker run --smoke-test`，四项均 exit 0 方写入出题侧 Docker 基线证明。
- SQLite 纯 Go 驱动，离线可构建；Dockerfile 已显式 `ENV GOPROXY=https://goproxy.cn,direct` 与 `ENV GOSUMDB=sum.golang.google.cn`。
