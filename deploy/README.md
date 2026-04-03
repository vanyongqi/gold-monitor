# Gold Monitor 自动部署说明

推荐复用你当前博客的链路：

1. 推送到 `main`
2. GitHub Actions 运行测试并构建镜像
3. 推送镜像到 `ghcr.io/<你的仓库>`
4. 云服务器定时拉取最新镜像
5. 如果镜像 digest 变化，则重建 `gold-monitor` 容器

## 线上建议约定

- 部署目录：`/opt/gold-monitor`
- 数据目录：`/opt/gold-monitor/storage`
- 线上容器名：`gold-monitor`
- 监听方式：容器 `8080` 绑定到宿主机 `127.0.0.1:18090`
- Nginx 再把 `https://718614413.xyz/gold/` 转发到 `127.0.0.1:18090`

## Nginx 路由示例

```nginx
location /gold/ {
    proxy_pass http://127.0.0.1:18090/;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
}
```

注意：

- `proxy_pass` 后面带结尾 `/`，这样 `/gold/...` 会被去掉前缀后再转给应用。
- 前端静态资源和接口已经改成相对路径，适配 `/gold/` 子路径部署。

## 服务器部署脚本

可参考 [deploy-ghcr.sh](/Users/fitz/personal/gold_monitor/deploy/gold/deploy-ghcr.sh)。

建议放到：

```bash
/opt/gold-monitor/deploy-ghcr.sh
```

## 定时任务

```cron
* * * * * flock -n /tmp/gold-monitor-deploy.lock /opt/gold-monitor/deploy-ghcr.sh >> /var/log/gold-monitor-deploy.log 2>&1
```

