# 哪吒面板

阶段： *alpha*

服务期状态监控，被动接收（非 node-exporter 那种主动拉取的方式。）

## 系统设计

C/S 采用 gRPC 通信，客户端通过添加主机生成的单独 Token 上报监控信息。因为不会做成多用户的，上报信息会储存到内存中，暂不提供历史数据统计。

- 首次连接：上报基本信息（系统、CPU基本信息），后面管理员可从客户端主动拉取更新。
- 监控上报：每隔 3s 向服务器上报系统信息

## 一键脚本

WIP，尚未完成，还在做监控端安装

```shell
curl -L https://raw.githubusercontent.com/naiba/nezha/master/script/install.sh -o nezha.sh && chmod +x nezha.sh
sudo nezha.sh
```
## FAQ

- 反代后打开面板提示「实时通道断开」：[https://www.google.com/search?q=nginx+%E5%8F%8D%E4%BB%A3+websocket](Nginx 反代 WebSocket)

## 社区文章

 - [哪吒面板：小鸡们的最佳探针](https://www.zhujizixun.com/2843.html) *（已过时）*
