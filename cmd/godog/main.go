package main

import (
	"flag"
	"log"

	_ "github.com/bingoohuang/godog/autoload"
	"github.com/dustin/go-humanize"
)

/*
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <time.h>

// C 函数：分配内存并用随机值填充
void *allocate_and_fill_random(size_t size) {
    void *mem = malloc(size);
    if (mem == NULL) {
        perror("Memory allocation failed");
        exit(EXIT_FAILURE);
    }

    // 初始化随机数生成器
    srand((unsigned int)time(NULL));

    // 填充随机值
	size_t i = 0;
    for (; i < size; ++i) {
        ((unsigned char *)mem)[i] = rand() % 256;
    }
    return mem;
}

// C 函数：释放内存
void free_memory(void *mem) {
    free(mem);
}
*/
import "C"

func main() {
	cgoMem := flag.String("cgo-mem", "", "cgo malloc memory size")
	flag.Parse()

	if *cgoMem != "" {
		cgoMemSize, err := humanize.ParseBytes(*cgoMem)
		if err != nil {
			log.Fatalf("cgo mem parse error: %v", err)
		}

		if cgoMemSize > 0 {
			mem := C.allocate_and_fill_random(C.size_t(cgoMemSize))
			if mem == nil {
				log.Fatalf("Failed to allocate memory")
			}
		}
	}

	select {}
}
