# godog

watch self's memory and cpu, check its threshold, and take actions

## Sample

- build: `make`
- run: `DOG_DEBUG=1 DOG_INTERVAL=3s DOG_RSS=30MiB DOG_CPU=200 godog` 每 3 秒检查一次, 内存上限 30 MiB, CPU 上限 200%
- busy: `jj mem=30MiB > Dog.busy` 打满 30 MiB 内存
- busy: ` jj cores:=3 cpu:=100 > Dog.busy` 打满 3 个核

## Usage

一行代码集成

`impott _ "github.com/bingoohuang/godog/autoload"`

## Environment

| Name         | Default    | Meaning                                                 | Usage                |
|--------------|------------|---------------------------------------------------------|----------------------|
| DOG_DEBUG    | 0          | Debug mode                                              | `export DOG_DEBUG=1` |
| DOG_RSS      | 256 MiB    | 内存上限 ｜ `export DOG_RSS=30MiB`                           |
| DOG_CPU      | 66 * cores | CPU百分比上限 ｜ `export DOG_CPU=200`                         |
| DOG_INTERVAL | 1m         | 检查时间间隔 ｜ `export DOG_INTERVAL=5m`                       |
| DOG_JITTER   | 10s        | 间隔补充随机时间｜ `export DOG_JITTER=1m`                        |
| DOG_TIMES    | 5          | 触发上限次数｜ `export DOG_TIMES=10`                           |
| DOG_DIR      | 当前目录       | 检查 Dog.busy 和生成 Dog.exit 的路径｜ `export DOG_DIR=/etc/dog` |

注:

- 达到次数默认会导致进程退出，保护整个系统
- 退出时，会生成文件 Dog.exit

Dog.exit 文件内容示例

`[RSS 连续 5 次超标 31457280: [31461376 31469568 31469568 31469568 31469568]]`

