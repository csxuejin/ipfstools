# IPFS Cluster 相关命令行工具

## 编译
- 运行 `make mac` 编译成 mac 下可执行文件 ipfstools
- 运行 `make linux` 编译成 linux 下可执行文件 ipfstools

## 命令

#### ./ipfstools add

该命令会对当前目录下的 `testfiles` 文件夹中的所有文件进行 `add` 操作，并且声称记录哈希值的文件 `filehashes`。

#### ./ipfstools pinadd 

该命令会对当前目录下的 `filehashes` 文件中记录的哈希值进行 `pin add` 操作。

#### ./ipfstools rmall

该命令会删除当前集群中所有 `pin` 记录。

#### ./ipfstools gc

该命令会进行 gc 回收操作，你也可以使用原生的 `ipfs repo gc` 命令。

## 配置文件说明

- 配置文件存放目录：与 ipfstools 同级。
- 配置文件名称： config.json
- 配置文件中各项说明如下：

``` go
type Config struct {
	AddFileWorkerNum    int `json:"add_file_worker_num"`  // 进行 add 操作的并发 worker 数目，默认为 10
    PinAddFileWorkerNum int `json:"pin_add_file_worker_num"` // 进行 pin add 操作的并发 worker 数目，默认为 10
	PinAddWaitTime      int `json:"pin_add_wait_time"`  // 每一个 pin add 操作的间隔时间，单位为分钟
}
```
