当前操作流程：

1、根目录"make clean"保持当前环境没有冲突的容器和web service

2、进入fixtures/执行"docker-compose up"将fabric网络启动

3、进入app/执行编译"go build ."

4、启动web service执行命令"./app start"

