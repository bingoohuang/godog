# godog

watch self's memory and cpu, check its threshold, and take actions

## Sample

- build: `make`
- run: `DOG_DEBUG=1 DOG_INTERVAL=3s DOG_RSS=20MiB DOG_CPU=60 godog` 每 3 秒检查一次, 内存上限 30 MiB, CPU 上限 200%
- busy: `echo '{"mem":"20MiB"}' > Dog.busy` 打满 30 MiB 内存
- busy: `echo '{"cores":3,"cpu":100}' > Dog.busy` 打满 3 个核
- watch: `watch 'ps aux | awk '\''NR==1 || /godog/ && !/awk/'\'''`
- pprofile: `go tool pprof -http=:8080 Dog.xxx.prof`

## Usage

一行代码集成

`import _ "github.com/bingoohuang/godog/autoload"`

## Environment

| Name         | Default    | Meaning                      | Usage                     |
|--------------|------------|------------------------------|---------------------------|
| DOG_DEBUG    | 0          | Debug mode                   | `export DOG_DEBUG=1`      |
| DOG_RSS      | 256 MiB    | 内存上限                         | `export DOG_RSS=30MiB`    |
| DOG_CPU      | 66 * cores | CPU百分比上限                     | `export DOG_CPU=200`      |
| DOG_INTERVAL | 1m         | 检查时间间隔                       | `export DOG_INTERVAL=5m`  |
| DOG_JITTER   | 10s        | 间隔补充随机时间                     | `export DOG_JITTER=1m`    |
| DOG_TIMES    | 5          | 触发上限次数                       | `export DOG_TIMES=10`     |
| DOG_DIR      | 当前目录       | 检查 Dog.busy 和生成 Dog.exit 的路径 | `export DOG_DIR=/etc/dog` |

注:

- 达到次数，默认动作会导致进程退出，保护整个系统
- 退出时，会生成文件 Dog.exit

Dog.exit 文件内容示例

```json
{
  "pid": 20,
  "time": "2024-08-07T09:27:35+08:00",
  "reasons": [
    {
      "type": "RSS",
      "reason": "连续 5 次超标",
      "values": [20975616, 20996096, 20996096, 20996096, 21000192],
      "threshold": 20971520,
      "profile": "Dog.mem.88251.prof"
    }
  ]
}
```

```json
{
  "pid": 20,
  "time": "2024-08-07T09:30:11+08:00",
  "reasons": [
    {
      "type": "CPU",
      "reason": "连续 5 次超标",
      "values": [61, 61, 67, 69, 71],
      "threshold": 60,
      "profile": "Dog.cpu.88344.prof"
    }
  ]
}
```

##  cgo memory

1. test cgo memory malloc `DOG_DEBUG=1 DOG_INTERVAL=3s DOG_RSS=20MiB DOG_CPU=20 godog -cgo-mem 20MiB`
