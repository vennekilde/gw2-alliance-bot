package main

import (
	"log"
	"os"
	"os/signal"

	"github.com/vennekilde/gw2-alliance-bot/internal"
	"go.uber.org/zap"
)

func init() {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()
	_ = zap.ReplaceGlobals(logger)
	zap.L().Info("replaced zap's global loggers")
}

func main() {
	discordToken := os.Getenv("discordToken")
	backendURL := os.Getenv("backendURL")
	backendToken := os.Getenv("backendToken")

	bot := internal.NewBot(discordToken, backendURL, backendToken)
	bot.Start()
	defer bot.Close()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop
	log.Println("Graceful shutdown")
}
