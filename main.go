package main

import (
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

func checkError(a cli.ActionFunc) cli.ActionFunc {
	return func(c *cli.Context) error {
		err := a(c)
		if err != nil {
			log.Fatal(err)
		}

		return nil
	}
}

func init() {
	log.SetOutput(os.Stdout)
}

func main() {

	app := cli.NewApp()
	app.Usage = "Vault plugin manager as side-container for each Vault instance"
	app.Description = "Vault plugin manager syncs plugins with S3 bucket in a side-container of each Vault instance"

	app.Commands = []cli.Command{
		{
			Name: "run",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "s3-bucket",
					Required: true,
				},
				&cli.StringFlag{
					Name:  "plugin-path",
					Value: "/vault/plugins",
				},
				&cli.DurationFlag{
					Name:  "interval",
					Value: 180 * time.Second,
				},
				&cli.StringFlag{
					Name:  "region",
					Value: "eu-central-1",
				},
				&cli.StringFlag{
					Name:     "vault-auth-path",
					Required: true,
				},
				&cli.StringFlag{
					Name:     "vault-auth-role",
					Required: true,
				},
				&cli.IntFlag{
					Name:  "vault-max-retries",
					Value: 8,
				},
				&cli.StringFlag{
					Name:  "sa-token-path",
					Value: "/var/run/secrets/kubernetes.io/serviceaccount/token",
				},
			},
			Action: checkError(run),
		},
	}

	app.Run(os.Args)
}
