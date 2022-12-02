// SPDX-License-Identifier: GPL-3.0-only

package main

import (
    "github.com/mzyy94/nscon"
    "log"
    "os"
    "os/exec"
    "os/signal"
    "time"
)

func setInput(input *uint8) {
    *input++
    time.AfterFunc(100*time.Millisecond, func() {
        *input--
    })
}

func main() {
    target := "/dev/hidg0"
    con := nscon.NewController(target)
    con.LogLevel = 2
    defer con.Close()
    con.Connect()

    // Set tty break for read keyboard input directly
    exec.Command("stty", "-F", "/dev/tty", "cbreak", "min", "1").Run()
    defer exec.Command("stty", "-F", "/dev/tty", "-cbreak").Run()
    exec.Command("stty", "-F", "/dev/tty", "-echo").Run()
    defer exec.Command("stty", "-F", "/dev/tty", "echo").Run()

    go func() {
        for {
            setInput(&con.Input.Button.A)
            log.Printf("%c: Press A!", time.Now())
            time.Sleep(time.Millisecond * 250)
        }
    }()

    c := make(chan os.Signal, 1)
    signal.Notify(c, os.Interrupt)

    select {
    case <-c:
        return
    }

}
