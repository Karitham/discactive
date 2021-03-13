package main

import (
	"os"

	"github.com/Karitham/discactive/disc"
	"github.com/Karitham/discactive/img"
	"github.com/rs/zerolog/log"

	/// Font & template
	_ "embed"
)

//go:embed assets/template.png
var tmpl []byte

//go:embed assets/inconsolata_regular.ttf
var ft []byte

// TODO: Clean up this mess
// eww
func init() {
	img.Init(tmpl, ft)
}

func main() {
	// Get token from ENV
	token, ok := os.LookupEnv("DISCORD_TOKEN")
	if !ok {
		log.Panic().Msg("could not find token in env")
	}

	bot := disc.New(token)

	// Load Users to track
	// Done here for testing purposes.
	// Probably through a PUT endpoint in the future

	if err := bot.LoadUsersFromJSON("assets/user_tracking.json"); err != nil {
		log.Panic().Err(err)
	}

	// Run the bot
	err := bot.RunWithEventChan()
	if err != nil {
		log.Panic().Err(err)
	}

	// Here we just range over the events and generate them
	// This is for convenience
	for id := range bot.Pres.Event {
		i := img.New(bot.Pres.Users[id])

		if _, err := i.Generate(); err != nil {
			log.Err(err).Msg("Error drawing imgage. Impossible for now")
			continue
		}
		log.Trace().Str("User", id.String()).Msg("Generated image")

		f, err := os.Create(bot.Pres.Users[id].User.Username + ".png")
		if err != nil {
			panic(err)
		}

		if err := i.To(f); err != nil {
			log.Err(err).Msg("Error writing to file")
		}

		f.Close()
	}

}
