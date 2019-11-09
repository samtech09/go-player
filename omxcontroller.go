// PLAYER OBJECT STRUCT AND METHODS
package main

import (
	"io"
	"os"
	"time"
	"os/exec"
	"syscall"
)

var commandList = map[string]string{"pause": "p", "up": "\x1b[A", "down": "\x1b[B", "left": "\x1b[D", "right": "\x1b[C", "volup": "+", "voldown": "-"}

type Player struct {
	Playing  bool
	Paused   string
	FilmName string
	Film     *exec.Cmd
	PipeIn   io.WriteCloser
}

func (p *Player) StartFilm(name string) error {
	var err error
	p.FilmName = name
	p.Paused = "Pause"
	p.Playing = true
	if config.Playmode == "keyboard" {
		p.Film = exec.Command("lxterminal", "-e", "omxplayer", "-o", "hdmi", name)
	} else {
		p.Film = exec.Command("omxplayer", "-o", "hdmi", name)
	}
	p.Film.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	p.PipeIn, err = p.Film.StdinPipe()
	if err == nil {
		p.Film.Stdout = os.Stdout
		err = p.Film.Start()
	}

	// wait for 2 sec
	if config.Playmode == "keyboard" {
		p.Playing = false
		time.Sleep(3 * time.Second)
	}

	return err
}

func (p *Player) PauseFilm() {
	if p.Paused == "Pause" {
		p.Paused = "Play"
	} else {
		p.Paused = "Pause"
	}
}

func (p *Player) EndFilm() error {
	pgid, err := syscall.Getpgid(p.Film.Process.Pid)
	if err == nil {
		syscall.Kill(-pgid, 15)
		p.FilmName = ""
		pageData.CurrentFilm = ""
		p.Playing = false
	}
	return err
}

func (p *Player) SendCommandToFilm(command string) error {
	if command == "pause" {
		p.PauseFilm()
	}
	_, err := p.PipeIn.Write([]byte(commandList[command]))
	return err
}
