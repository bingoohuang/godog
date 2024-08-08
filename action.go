package godog

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"time"
)

type Action interface {
	DoAction(dir string, debug bool, reasons []ReasonItem)
}

type ActionFn func(dir string, debug bool, reasons []ReasonItem)

func (f ActionFn) DoAction(dir string, debug bool, reasons []ReasonItem) {
	f(dir, debug, reasons)
}

type ExitFile struct {
	Pid     int          `json:"pid"`
	Time    string       `json:"time"`
	Reasons []ReasonItem `json:"reasons"`
}

const DogExit = "Dog.exit"

var DefaultAction = func(dir string, debug bool, reasons []ReasonItem) {
	log.Printf("program exit by godog, reason: %v", reasons)

	data, _ := json.Marshal(ExitFile{
		Pid:     os.Getpid(),
		Time:    time.Now().Format(time.RFC3339),
		Reasons: reasons,
	})

	name := filepath.Join(dir, DogExit)
	_ = os.WriteFile(name, data, os.ModePerm)
	os.Exit(1)
}
