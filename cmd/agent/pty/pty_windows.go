//go:build windows

package pty

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/artdarek/go-unzip"
	"github.com/iamacarpet/go-winpty"
)

type Pty struct {
	tty *winpty.WinPTY
}

func DownloadDependency() {
	executablePath, err := getExecutableFilePath()
	if err != nil {
		fmt.Println("NEZHA>> wintty 获取文件路径失败", err)
		return
	}

	winptyAgentExe := filepath.Join(executablePath, "winpty-agent.exe")
	winptyAgentDll := filepath.Join(executablePath, "winpty.dll")

	fe, errFe := os.Stat(winptyAgentExe)
	fd, errFd := os.Stat(winptyAgentDll)
	if errFe == nil && fe.Size() > 300000 && errFd == nil && fd.Size() > 300000 {
		return
	}

	resp, err := http.Get("https://dn-dao-github-mirror.daocloud.io/rprichard/winpty/releases/download/0.4.3/winpty-0.4.3-msvc2015.zip")
	if err != nil {
		log.Println("NEZHA>> wintty 下载失败", err)
		return
	}
	defer resp.Body.Close()
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("NEZHA>> wintty 下载失败", err)
		return
	}
	if err := os.WriteFile("./wintty.zip", content, os.FileMode(0777)); err != nil {
		log.Println("NEZHA>> wintty 写入失败", err)
		return
	}
	if err := unzip.New("./wintty.zip", "./wintty").Extract(); err != nil {
		fmt.Println("NEZHA>> wintty 解压失败", err)
		return
	}
	arch := "x64"
	if runtime.GOARCH != "amd64" {
		arch = "ia32"
	}

	os.Rename("./wintty/"+arch+"/bin/winpty-agent.exe", winptyAgentExe)
	os.Rename("./wintty/"+arch+"/bin/winpty.dll", winptyAgentDll)
	os.RemoveAll("./wintty")
	os.RemoveAll("./wintty.zip")
}

func getExecutableFilePath() (string, error) {
	ex, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(ex), nil
}

func Start() (*Pty, error) {
	shellPath, err := exec.LookPath("powershell.exe")
	if err != nil || shellPath == "" {
		shellPath = "cmd.exe"
	}
	path, err := getExecutableFilePath()
	if err != nil {
		return nil, err
	}
	tty, err := winpty.OpenDefault(path, shellPath)
	return &Pty{tty: tty}, err
}

func (pty *Pty) Write(p []byte) (n int, err error) {
	return pty.tty.StdIn.Write(p)
}

func (pty *Pty) Read(p []byte) (n int, err error) {
	return pty.tty.StdOut.Read(p)
}

func (pty *Pty) Setsize(cols, rows uint32) error {
	pty.tty.SetSize(cols, rows)
	return nil
}

func (pty *Pty) Close() error {
	pty.tty.Close()
	return nil
}
