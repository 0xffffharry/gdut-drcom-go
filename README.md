## gdut-drcom-go

    gdut-drcom 的 Golang 实现

    用法：
    1. 安装Golang环境以及make
    2. make build

    参数说明：
    -i string 远程IP
    -p uint16 远程端口
    -b string 绑定本地接口
    -a string 绑定接口IP而不是接口名
    -f string 日志文件
    //
    -c string 配置文件（支持多认证）（JSON）
    {
        "log_file": "",
        "debug": false,
        "core": [
            {
                "tag": "test" // 标识
                "remote_ip": "10.0.3.6", // 远程IP
                "remote_port": 61440, // 远程端口
                "bind_device": "pppoe-wan", // 绑定本地接口
                "bind_to_addr": true // 绑定接口IP而不是接口名
            }
            ...
        ]
    }