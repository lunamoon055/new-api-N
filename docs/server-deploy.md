# 服务器上线部署说明

这套部署方式适合你现在的目标：服务器上稳定跑生产环境，本机继续改代码；每次改完以后，在本机执行一个脚本同步到服务器并重建线上容器。

## 服务器要求

- Linux 服务器，推荐 Ubuntu 22.04/24.04。
- 已安装 Docker 和 Docker Compose v2。
- 防火墙/安全组放行 `80` 和 `443`。
- 如果要 HTTPS，域名 DNS 的 A 记录已经指向服务器 IP。
- 服务器能访问 Docker Hub 和 Go 依赖源；前端资源会先在本机用 Bun 构建好，再同步到服务器。
- 本机已安装 Bun。脚本会自动查找 `bun` 或 `~/.bun/bin/bun`，也可以用 `BUN_BIN=/path/to/bun` 指定。

## 首次上线

先在本机保存服务器连接信息：

```bash
cp .deploy.env.example .deploy.env
```

然后编辑 `.deploy.env`：

```env
NEW_API_HOST=root@你的服务器IP
NEW_API_SSH_PORT=
NEW_API_PATH=/opt/new-api
PUBLIC_URL=https://你的域名
```

`.deploy.env` 只保存在你的电脑上，已被 `.gitignore` 忽略，不会同步进代码仓库。

接着生成生产环境密钥：

```bash
make prod-env
```

这会生成 `.env.production`，里面包含随机数据库密码、Redis 密码、会话密钥和加密密钥。生成后可以打开检查：

```bash
sed -n '1,120p' .env.production
```

然后部署到服务器：

```bash
make prod-deploy
```

如果不想使用 `.deploy.env`，也可以临时写在命令前：

```bash
NEW_API_HOST=root@你的服务器IP NEW_API_SSH_PORT=2222 NEW_API_PATH=/opt/new-api ./scripts/deploy-remote.sh
```

部署完成后访问：

```text
https://你的域名
```

第一次打开会进入初始化向导，按页面提示创建管理员账号。

## 没有域名时

如果暂时只想用服务器 IP 测试，可以生成：

```bash
./scripts/create-production-env.sh http://你的服务器IP
```

然后把 `.env.production` 里的 `PUBLIC_HOST` 改为：

```env
PUBLIC_HOST=:80
TRUSTED_REDIRECT_DOMAINS=你的服务器IP
```

再执行部署脚本。这样会先用 HTTP 访问：

```text
http://你的服务器IP
```

## 后续修改和更新

你以后在本机修改代码后，先做本地检查，再执行同一个部署命令：

```bash
make prod-deploy
```

脚本会：

1. 在本机用 Bun 构建 `web/default/dist` 和 `web/classic/dist`。
2. 用 `rsync` 同步当前项目和前端 dist 到服务器。
3. 保留服务器上的 `.env.production`、数据库数据、日志和构建缓存目录。
4. 检查服务器上 Docker、Docker Compose 和 Docker daemon 是否可用。
5. 在服务器执行 `docker compose --env-file .env.production -f docker-compose.prod.yml up -d --build`。
6. 输出线上容器状态。

如果你确实要把本机 `.env.production` 覆盖到服务器：

```bash
NEW_API_SYNC_ENV=1 NEW_API_HOST=root@你的服务器IP NEW_API_PATH=/opt/new-api ./scripts/deploy-remote.sh
```

## 常用运维命令

进入服务器后：

```bash
cd /opt/new-api
docker compose --env-file .env.production -f docker-compose.prod.yml ps
docker compose --env-file .env.production -f docker-compose.prod.yml logs -f new-api
docker compose --env-file .env.production -f docker-compose.prod.yml logs -f caddy
docker compose --env-file .env.production -f docker-compose.prod.yml restart new-api
```

也可以直接在本机执行：

```bash
make prod-ps
make prod-logs
make prod-restart
```

查看其他服务日志时指定 `SERVICE`：

```bash
SERVICE=caddy make prod-logs
```

停止线上服务：

```bash
cd /opt/new-api
docker compose --env-file .env.production -f docker-compose.prod.yml down
```

备份 PostgreSQL：

```bash
cd /opt/new-api
docker compose --env-file .env.production -f docker-compose.prod.yml exec -T postgres pg_dump -U newapi new_api > backup-new-api.sql
```

恢复 PostgreSQL：

```bash
cd /opt/new-api
cat backup-new-api.sql | docker compose --env-file .env.production -f docker-compose.prod.yml exec -T postgres psql -U newapi -d new_api
```

## 本机开发建议

后端和前端开发仍然用项目已有命令：

```bash
make dev-api
make dev-web
```

如果你只想启动默认前端：

```bash
cd web/default
bun install
bun run dev
```

上线前至少做一次生产配置检查：

```bash
docker compose --env-file .env.production -f docker-compose.prod.yml config
```

## 需要你提供的信息

我真正帮你执行上线时，需要下面三项：

```text
服务器 SSH：例如 root@1.2.3.4
SSH 端口：默认 22，如果不是请告诉我
域名：例如 api.example.com；如果暂时没有域名，给服务器公网 IP 也可以
```
