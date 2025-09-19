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
					client, err := NewClient(
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
					time.Sleep(
						2 * time.Second,
					) // give the underlying system some extra time to push the data out
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
					srv := NewudpServer(
						cmd.String("host"),
						int(cmd.Int32("port")),
						cmd.String("file"),
					)
					if err := srv.Start(ctx); err != nil {
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
