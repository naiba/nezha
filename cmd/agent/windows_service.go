//go:build windows
// +build windows

package main

//允许使用 sc.exe Create "nezha-agent" binPath= "D:\git\nezha\agent.exe -s 114.230.160.114:5555 -p e86903818636756455" 的方式创建windows服务，不依赖nssm
// Copypasta from the example files:
// https://github.com/golang/sys/blob/master/windows/svc/example

import (
	"fmt"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/debug"
	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc/mgr"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	windowsServiceName        = "nezha-agent"
	windowsServiceDescription = "nezha-agent"
)

func controlServiceStop() error {
	return controlService(svc.Stop, svc.Stopped)
}
func controlServicePause() error {
	return controlService(svc.Pause, svc.Paused)
}
func controlServiceContinue() error {
	return controlService(svc.Continue, svc.Running)
}
func exePath() (string, error) {
	prog := os.Args[0]
	p, err := filepath.Abs(prog)
	if err != nil {
		return "", err
	}
	fi, err := os.Stat(p)
	if err == nil {
		if !fi.Mode().IsDir() {
			return p, nil
		}
		err = fmt.Errorf("%s is directory", p)
	}
	if filepath.Ext(p) == "" {
		p += ".exe"
		fi, err := os.Stat(p)
		if err == nil {
			if !fi.Mode().IsDir() {
				return p, nil
			}
			err = fmt.Errorf("%s is directory", p)
		}
	}
	return "", err
}

//func installWindowsService() error {
//	exepath, err := os.Executable()
//	if err != nil {
//		return errors.Wrap(err, "Cannot find path name that start the process")
//	}
//	m, err := mgr.Connect()
//	if err != nil {
//		return errors.Wrap(err, "Cannot establish a connection to the service control manager")
//	}
//	defer m.Disconnect()
//	s, err := m.OpenService(windowsServiceName)
//	if err != nil {
//		return err
//	}
//	//extraArgs, err := getServiceExtraArgsFromCliArgs(c, &log)
//	//if err != nil {
//	//	errMsg := "Unable to determine extra arguments for windows service"
//	//	return errors.Wrap(err, errMsg)
//	//}
//
//	config := mgr.Config{StartType: mgr.StartAutomatic, DisplayName: windowsServiceDescription}
//	s, err = m.CreateService(windowsServiceName, exepath, config)
//	if err != nil {
//		return errors.Wrap(err, "Cannot install service")
//	}
//	defer s.Close()
//	err = eventlog.InstallAsEventCreate(windowsServiceName, eventlog.Error|eventlog.Warning|eventlog.Info)
//	if err != nil {
//		s.Delete()
//		return errors.Wrap(err, "Cannot install event logger")
//	}
//
//	err = configRecoveryOption(s.Handle)
//	if err != nil {
//		return err
//	}
//
//	err = s.Start()
//	return err
//}
//
//func uninstallWindowsService() error {
//	m, err := mgr.Connect()
//	if err != nil {
//		return errors.Wrap(err, "Cannot establish a connection to the service control manager")
//	}
//	defer m.Disconnect()
//	s, err := m.OpenService(windowsServiceName)
//	if err != nil {
//		return fmt.Errorf("Agent service %s is not installed, so it could not be uninstalled", windowsServiceName)
//	}
//	defer s.Close()
//
//	if status, err := s.Query(); err == nil && status.State == svc.Running {
//		if _, err := s.Control(svc.Stop); err != nil {
//			return err
//		}
//	}
//
//	err = s.Delete()
//	if err != nil {
//		return errors.Wrap(err, "Cannot delete agent service")
//	}
//	err = eventlog.Remove(windowsServiceName)
//	if err != nil {
//		return errors.Wrap(err, "Cannot remove event logger")
//	}
//	return nil
//}
//
//// https://msdn.microsoft.com/en-us/library/windows/desktop/ms685937(v=vs.85).aspx
//// Not supported in Windows Server 2003 and Windows XP
//type serviceFailureActionsFlag struct {
//	// enableActionsForStopsWithErr is of type BOOL, which is declared as
//	// typedef int BOOL in C
//	enableActionsForStopsWithErr int
//}
//
//type recoveryAction struct {
//	recoveryType uint32
//	// The time to wait before performing the specified action, in milliseconds
//	delay uint32
//}
//
//// until https://github.com/golang/go/issues/23239 is release, we will need to
//// configure through ChangeServiceConfig2
//func configRecoveryOption(handle windows.Handle) error {
//	actions := []recoveryAction{
//		{recoveryType: uint32(scActionRestart), delay: uint32(recoverActionDelay / time.Millisecond)},
//	}
//	serviceRecoveryActions := serviceFailureActions{
//		resetPeriod: uint32(failureCountResetPeriod / time.Second),
//		actionCount: uint32(len(actions)),
//		actions:     uintptr(unsafe.Pointer(&actions[0])),
//	}
//	if err := windows.ChangeServiceConfig2(handle, windows.SERVICE_CONFIG_FAILURE_ACTIONS, (*byte)(unsafe.Pointer(&serviceRecoveryActions))); err != nil {
//		return err
//	}
//	serviceFailureActionsFlag := serviceFailureActionsFlag{enableActionsForStopsWithErr: 1}
//	return windows.ChangeServiceConfig2(handle, serviceConfigFailureActionsFlag, (*byte)(unsafe.Pointer(&serviceFailureActionsFlag)))
//}
func installService() error {
	exepath, err := exePath()
	if err != nil {
		return err
	}
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()
	s, err := m.OpenService(windowsServiceName)
	if err == nil {
		s.Close()
		return fmt.Errorf("service %s already exists", windowsServiceName)
	}
	//TODO 这里的启动参数无法获取，不知道为什么
	s, err = m.CreateService(windowsServiceName, exepath, mgr.Config{DisplayName: windowsServiceDescription}, "-s=67.230.160.114:5555", "-p=e86903818636756465")
	if err != nil {
		return err
	}
	defer s.Close()
	err = eventlog.InstallAsEventCreate(windowsServiceName, eventlog.Error|eventlog.Warning|eventlog.Info)
	if err != nil {
		s.Delete()
		return fmt.Errorf("SetupEventLogSource() failed: %s", err)
	}
	return nil
}
func removeService() error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()
	s, err := m.OpenService(windowsServiceName)
	if err != nil {
		return fmt.Errorf("service %s is not installed", windowsServiceName)
	}
	defer s.Close()
	err = s.Delete()
	if err != nil {
		return err
	}
	err = eventlog.Remove(windowsServiceName)
	if err != nil {
		return fmt.Errorf("RemoveEventLogSource() failed: %s", err)
	}
	return nil
}
func startService() error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()
	s, err := m.OpenService(windowsServiceName)
	if err != nil {
		return fmt.Errorf("could not access service: %v", err)
	}
	defer s.Close()
	err = s.Start("is", "manual-started")
	if err != nil {
		return fmt.Errorf("could not start service: %v", err)
	}
	return nil
}

func controlService(c svc.Cmd, to svc.State) error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()
	s, err := m.OpenService(windowsServiceName)
	if err != nil {
		return fmt.Errorf("could not access service: %v", err)
	}
	defer s.Close()
	status, err := s.Control(c)
	if err != nil {
		return fmt.Errorf("could not send control=%d: %v", c, err)
	}
	timeout := time.Now().Add(10 * time.Second)
	for status.State != to {
		if timeout.Before(time.Now()) {
			return fmt.Errorf("timeout waiting for service to go to state=%d", to)
		}
		time.Sleep(300 * time.Millisecond)
		status, err = s.Query()
		if err != nil {
			return fmt.Errorf("could not retrieve service status: %v", err)
		}
	}
	return nil
}

var elog debug.Log

type myservice struct{}

func (m *myservice) Execute(serviceArgs []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue
	changes <- svc.Status{State: svc.StartPending}
	elog.Info(1, "osArgs "+strings.Join(os.Args, ","))
	//errC := make(chan error)
	go func() {
		run()
	}()
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
loop:
	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
				// Testing deadlock from https://code.google.com/p/winsvc/issues/detail?id=4
				time.Sleep(100 * time.Millisecond)
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				break loop
			case svc.Pause:
				changes <- svc.Status{State: svc.Paused, Accepts: cmdsAccepted}
			case svc.Continue:
				changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
			default:
				elog.Error(1, fmt.Sprintf("unexpected control request #%d", c))
			}
		}
	}
	changes <- svc.Status{State: svc.StopPending}
	return
}
func runService(isDebug bool) {
	var err error
	if isDebug {
		elog = debug.New(windowsServiceName)
	} else {
		elog, err = eventlog.Open(windowsServiceName)
		if err != nil {
			return
		}
	}
	defer elog.Close()

	elog.Info(1, fmt.Sprintf("starting %s service", windowsServiceName))
	run := svc.Run
	if isDebug {
		run = debug.Run
	}
	err = run(windowsServiceName, &myservice{})
	if err != nil {
		elog.Error(1, fmt.Sprintf("%s service failed: %v", windowsServiceName, err))
		return
	}
	elog.Info(1, fmt.Sprintf("%s service stopped", windowsServiceName))
}
func runSvc() {
	inService, err := svc.IsWindowsService()
	if err != nil {
		log.Fatalf("failed to determine if we are running in service: %v", err)
	}
	//使用sc创建的服务
	if inService {
		runService(false)
		return
	}
	run()
}
