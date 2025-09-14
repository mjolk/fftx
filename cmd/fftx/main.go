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
						Name:    "file",
						Value:   `/home/mjolk/Downloads/coh3.part1.rar`,
						Usage:   "file",
						Aliases: []string{"f"},
					},
					&cli.StringFlag{
						Name:    "ip",
						Value:   `192.168.50.198`,
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
						cmd.String("file"),
						cmd.String("ip"),
						int(cmd.Int32("port")),
					)
					if err != nil {
						return err
					}
					sent, err := client.Send()
					if err != nil {
						return err
					}
					log.Printf(
						"sent: %d  bytes \n",
						sent,
					)
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
						Value:   `192.168.50.198`,
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
						Name:    "file",
						Value:   `coh.rar`,
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
					srv := fftx.NewudpServer(
						cmd.String("host"),
						int(cmd.Int32("port")),
						cmd.String("file"),
					)
					if err := srv.Start(); err != nil {
						return err
					}
					received, written, err := srv.Recv(ctx)
					if err != nil {
						return err
					}

					log.Printf(
						"received: %d bytes, written %d bytes \n",
						received,
						written,
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
