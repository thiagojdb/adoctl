package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

type WatchConfig struct {
	IntervalSec int
	RefreshFunc func() error
	ClearScreen func()
	OnError     func(error)
	OnTick      func()
	refreshFunc func() error
	clearScreen func()
}

func NewWatchCommand(name, short, long string, cfg WatchConfig) *cobra.Command {
	return &cobra.Command{
		Use:   name,
		Short: short,
		Long:  long,
		RunE: func(cmd *cobra.Command, args []string) error {
			intervalSec := 30
			if cmd.Flags().Changed("interval") {
				intervalSec, _ = cmd.Flags().GetInt("interval")
			}
			return RunWatch(WatchConfig{
				IntervalSec: intervalSec,
				RefreshFunc: cfg.RefreshFunc,
				ClearScreen: cfg.ClearScreen,
				OnError:     cfg.OnError,
				OnTick:      cfg.OnTick,
			})
		},
	}
}

func RunWatch(cfg WatchConfig) error {
	interval := time.Duration(cfg.IntervalSec) * time.Second
	if interval <= 0 {
		interval = 30 * time.Second
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		if cfg.ClearScreen != nil {
			cfg.ClearScreen()
		}

		if cfg.RefreshFunc != nil {
			if err := cfg.RefreshFunc(); err != nil {
				if cfg.OnError != nil {
					cfg.OnError(err)
				}
			}
		}

		if cfg.OnTick != nil {
			cfg.OnTick()
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func ClearScreen() {
	fmt.Print("\033[H\033[2J")
}
