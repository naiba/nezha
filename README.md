# 哪吒面板

阶段： *alpha*

服务期状态监控，被动接收（非 node-exporter 那种主动拉取的方式。）

## 基本设计

### 用户系统

- GitHub 登录

### 通信

C/S 采用 gRPC 通信，客户端通过添加主机生成的单独 Token 上报监控信息。因为不会做成多用户的，上报信息会储存到内存中，暂不提供历史数据统计。

- 首次连接：上报基本信息（系统、CPU基本信息），后面管理员可从客户端主动拉取更新。
- 监控上报：每隔 3s 向服务器上报系统信息

配置文件参考：

```yaml
debug: true
httpport: 80
github:
  admin:
    - 用户 ID，看自己 GitHub 头像链接后面那一串数字
  clientid: GitHub Oauth App clientID
  clientsecret: client secret
site:
  brand: 站点标题
  cookiename: tulong #Cookie 名
```
