package service

import (
	"github.com/kardianos/service"
	"log"
	"os"
	"time"
)

//允许使用 sc Create "nezha-agent" start= auto binPath= "D:\git\nezha\agent.exe -s 114.230.160.114:5555 -p e86903818636756455" 的方式创建windows服务，不依赖nssm。如果有空格，另外处理。
//net start/stop nezha-agent
//sc delete nezha-agent

const (
	windowsServiceName        = "nezha-agent"
	windowsServiceDescription = "nezha-agent"
)

var subCommands = map[string]bool{
	"start":     true,
	"stop":      true,
	"restart":   true,
	"install":   true,
	"uninstall": true,
}

type Program struct {
	runFunc   func()
	svc       service.Service
	svcLogger service.Logger
}

func (p *Program) Start(s service.Service) error {
	go func() {
		for {
			func() {
				if r := recover(); r != nil {
					p.svcLogger.Errorf("Recovering from panic in runService error is: %v", r)
					time.Sleep(10 * time.Second)
				}
				p.runFunc()
			}()
		}
	}()
	return nil
}
func (p *Program) Stop(s service.Service) error {
	return nil
}

func (prg *Program) RunService() {
	logger, err := prg.svc.Logger(nil)
	if err != nil {
		log.Fatal(err)
	}
	prg.svcLogger = logger
	err = prg.svc.Run()
	if err != nil {
		logger.Error(err)
	}
}
func (prg *Program) ConfigService(runFunc func()) {
	prg.runFunc = runFunc
	svcConfig := &service.Config{
		Name:        windowsServiceName,
		DisplayName: windowsServiceName,
		Description: windowsServiceDescription,
	}

	s, err := service.New(prg, svcConfig)
	if err != nil {
		log.Fatal(err)
	}
	if len(os.Args) > 1 {
		subCommand := os.Args[1]
		//只针对特定操作进行过滤
		if subCommands[subCommand] {
			err = service.Control(s, subCommand)
			if err != nil {
				log.Fatal(err)
			}
			return
		}
	}
	prg.svc = s
}
