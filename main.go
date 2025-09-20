package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/urfave/cli/v3"
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
						Name:    "file",
						Value:   ``,
						Usage:   "file",
						Aliases: []string{"f"},
					},
					&cli.StringFlag{
						Name:    "host",
						Value:   ``,
						Usage:   "host to send to",
						Aliases: []string{"h"},
					},
					&cli.Int32Flag{
						Aliases: []string{"p"},
						Name:    "port",
						Value:   8899,
						Usage:   "port",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					send := NewSend(
						cmd.String("file"),
						cmd.String("host"),
						int(cmd.Int32("port")),
					)

					if err := send.Start(ctx); err != nil {
						return err
					}

					time.Sleep(
						1 * time.Second,
					) // give the other side time to configure

					written, err := send.Send(ctx)
					if err != nil {
						return err
					}

					log.Printf("Done, sent %d MB", written/1024)
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
						Value:   ``,
						Usage:   "host",
						Aliases: []string{"h"},
					},
					&cli.StringFlag{
						Name:    "file",
						Value:   ``,
						Usage:   "file path",
						Aliases: []string{"f"},
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
					receive := NewReceiver(
						cmd.String("file"),
						cmd.String("host"),
						int(cmd.Int32("port")),
					)

					if err := receive.Start(ctx); err != nil {
						return err
					}

					read, written, err := receive.Recv(ctx)
					if err != nil {
						return err
					}

					log.Printf(
						"Done, read: %d MB, written: %d MB \n",
						read/1024,
						written/1024,
					)
					return nil
				},
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
