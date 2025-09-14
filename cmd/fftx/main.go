package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/urfave/cli/v3"

	"gitlab.com/astrotit/fftx/pkg/fftx"
)

func main() {
	cmd := &cli.Command{
		Commands: []*cli.Command{
			{
				Name:    "send",
				Aliases: []string{"s"},
				Usage:   "send file",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "path",
						Value:   ``,
						Usage:   "file",
						Aliases: []string{"p"},
					},
					&cli.StringFlag{
						Name:    "ip",
						Value:   ``,
						Usage:   "ip to send to",
						Aliases: []string{"i"},
					},
					&cli.Int32Flag{
						Aliases: []string{"p"},
						Name:    "port",
						Value:   8899,
						Usage:   "port",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					client, err := fftx.NewClient(
						cmd.String("path"),
						cmd.String("ip"),
						int(cmd.Int32("port")),
					)
					if err != nil {
						return err
					}
					if err := client.Send(); err != nil {
						return err
					}
					time.Sleep(8 * time.Second)
					return nil
				},
			},
			{
				Name:    "recv",
				Aliases: []string{"r"},
				Usage:   "receive file",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "host",
						Value:   `127.0.0.1`,
						Usage:   "host",
						Aliases: []string{"h"},
					},
					&cli.StringFlag{
						Name:    "source",
						Value:   `udp:\\blabla:3245`,
						Usage:   "allowed source ip",
						Aliases: []string{"s"},
					},
					&cli.StringFlag{
						Name:    "path",
						Value:   `C:\Pogram files\filename`,
						Usage:   "file path",
						Aliases: []string{"p"},
					},
					&cli.Int32Flag{
						Aliases: []string{"p"},
						Name:    "port",
						Value:   8899,
						Usage:   "port",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					fmt.Println("Starting file recv: ", cmd.String("path"))
					srv := fftx.NewudpServer(
						cmd.String("host"),
						int(cmd.Int32("port")),
						cmd.String("path"),
					)
					if err := srv.Start(); err != nil {
						return err
					}
					srv.Recv(ctx)
					return nil
				},
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
