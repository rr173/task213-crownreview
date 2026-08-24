# 评测构建用（与 Dockerfile 同源，供 build_benzhi_docker.sh 引用）
FROM docker.m.daocloud.io/library/golang:1.26.3-bookworm

ENV CGO_ENABLED=0
ENV GOTOOLCHAIN=local
ENV GOPROXY=https://goproxy.cn,direct
ENV GOSUMDB=sum.golang.google.cn
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOTOOLCHAIN=local go build -o /app/crownreview ./cmd/crownreview

ENTRYPOINT ["/app/crownreview"]
CMD ["--smoke-test"]
