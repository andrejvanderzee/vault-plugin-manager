package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

func run(c *cli.Context) error {

	// Create stop-channel to trap SIGTERM and prevent interruption of syncPlugins().
	stop := make(chan bool)
	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGTERM)

		for {
			sig := <-ch
			switch sig {
			case syscall.SIGTERM:
				stop <- true
			}
		}
	}()

	syncPlugins(c)

	// Execute syncPlugins() every interval until SIGTERM.
	ticker := time.NewTicker(c.Duration("interval"))
	for {
		select {
		case <-ticker.C:
			syncPlugins(c)
		case <-stop:
			ticker.Stop()
			return nil
		}
	}
}

// syncPlugins() is fail-safe, i.e. it handles errors without abort.
func syncPlugins(c *cli.Context) {

	s, err := newPluginSyncSession(c)
	if err != nil {
		log.Errorf("Failed to create plugin sync session: %s", err)
	} else if err = s.syncPlugins(); err != nil {
		log.Errorf("Failed to sync plugins: %s", err)
	}

	log.Infof("Next sync in about %s", c.String("interval"))
}
