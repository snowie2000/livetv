# 安装

## Windows
从 https://github.com/snowie2000/livetv/releases 下载livetv.exe，双击运行即可

## Linux
从 https://github.com/snowie2000/livetv/releases 下载livetv_amd64，`chmod +x livetv_amd64`，然后使用 `./livetv_amd64` 运行即可

## 更改端口
程序默认监听0.0.0.0:9000端口，如果需要更改监听地址或端口，可以使用`-listen`参数，例如 `./livetv_amd64 -listen 0.0.0.0:80`

## 密码
程序的默认密码为password，可以在后台直接修改。

如果忘记了设置的密码，可以在退出程序后使用 `-reset` 参数直接重置密码，例如 `./livetv_amd64 -reset 123456` 即可重置密码为123456


---

下一章： [开始使用](Usage_cn.md)