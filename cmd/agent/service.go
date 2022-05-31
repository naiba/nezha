package main

import (
	"github.com/kardianos/service"
	"log"
)

//允许使用 sc Create "nezha-agent" start= auto binPath= "D:\git\nezha\agent.exe -s 114.230.160.114:5555 -p e86903818636756455" 的方式创建windows服务，不依赖nssm。如果有空格，另外处理。
//net start/stop nezha-agent
//sc delete nezha-agent

const (
	windowsServiceName        = "nezha-agent"
	windowsServiceDescription = "nezha-agent"
)

var logger service.Logger

type program struct {
	funFunc func()
	svc     service.Service
}

func (p *program) Start(s service.Service) error {
	go p.run()
	return nil
}
func (p *program) run() {
	p.funFunc()
}
func (p *program) Stop(s service.Service) error {
	return nil
}

func (prg *program) runService() {
	logger, err := prg.svc.Logger(nil)
	if err != nil {
		log.Fatal(err)
	}
	err = prg.svc.Run()
	if err != nil {
		logger.Error(err)
	}
}
func (prg *program) configService(runFunc func()) {
	prg.funFunc = runFunc
	svcConfig := &service.Config{
		Name:        windowsServiceName,
		DisplayName: windowsServiceName,
		Description: windowsServiceDescription,
	}

	s, err := service.New(prg, svcConfig)
	if err != nil {
		log.Fatal(err)
	}
	prg.svc = s
}
