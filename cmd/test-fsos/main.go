package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"

	"github.com/annetutil/gnetcli/pkg/cmd"
	gcred "github.com/annetutil/gnetcli/pkg/credentials"
	"github.com/annetutil/gnetcli/pkg/device/fsos"
	"github.com/annetutil/gnetcli/pkg/streamer/ssh"
)

func main() {
	host, login, password := os.Args[1], os.Args[2], os.Args[3]
	logger := zap.NewNop()
	creds := gcred.NewSimpleCredentials(
		gcred.WithUsername(login),
		gcred.WithPassword(gcred.Secret(password)),
		gcred.WithLogger(logger),
	)
	connector := ssh.NewStreamer(host, creds, ssh.WithLogger(logger), ssh.WithPort(22))
	dev := fsos.NewDevice(connector)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	if err := dev.Connect(ctx); err != nil {
		fmt.Println("CONNECT ERROR:", err)
		os.Exit(1)
	}
	defer connector.Close()
	for _, c := range []string{"show version", "show clock"} {
		res, err := dev.Execute(cmd.NewCmd(c))
		if err != nil {
			fmt.Printf("=== %s === EXEC ERROR: %v\n", c, err)
			continue
		}
		fmt.Printf("=== %s (status=%d, out=%dB, err=%dB) ===\n%s\n", c, res.Status(), len(res.Output()), len(res.Error()), string(res.Output()))
	}
}
