package cmd

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"ezreal.com.cn/ez_river/mysql-http/river"
	"github.com/spf13/cobra"
)

var (
	// Used for flags.
	riverCfgFile string
)

// NewRiverCmd ...
func NewRiverCmd() *cobra.Command {
	var riverCmd = &cobra.Command{
		Use:   "river [string to echo]",
		Short: "river ",
		Run:   runRiver,
	}

	riverCmd.PersistentFlags().StringVar(&riverCfgFile, "riverCfgFile", "./river.toml", "config file (default is $HOME/river.toml)")
	return riverCmd
}

func runRiver(cmd *cobra.Command, args []string) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	cfg, err := river.NewConfigWithFile(riverCfgFile)
	if err != nil {
		log.Fatal("read file failed:", err)
	}
	r, err := river.NewRiver(cfg)
	if err != nil {
		log.Fatal(err)
	}
	if r == nil {
		log.Fatal("no river defined")
	}

	defer r.Close()
	go func() {
		r.Run()
	}()

	<-stop
	log.Println("\nShutting down the server...")
	log.Println("Server gracefully stopped")
}
