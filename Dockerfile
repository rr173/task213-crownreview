# 单阶段构建：golang 1.26.3，CGO 关闭用纯 Go SQLite 驱动
FROM docker.m.daocloud.io/library/golang:1.26.3-bookworm

ENV CGO_ENABLED=0
ENV GOTOOLCHAIN=local
ENV GOPROXY=https://goproxy.cn,direct
ENV GOSUMDB=sum.golang.google.cn
ENV GOFLAGS=-mod=mod

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOTOOLCHAIN=local go build -o /out/crownreview ./cmd/crownreview

WORKDIR /app
RUN cp /out/crownreview /app/crownreview

# 入口：默认执行 --smoke-test（Docker 双架构验证的唯一判据）；评测可覆盖为长驻服务
ENTRYPOINT ["/app/crownreview"]
CMD ["--smoke-test"]
