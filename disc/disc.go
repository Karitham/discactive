package disc

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/rs/zerolog/log"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/diamondburned/arikawa/v2/session"
)

// Bot is a discord bot
type Bot struct {
	Sess *session.Session
	Pres *Presence
}

// Presence represents the presence handler
type Presence struct {
	Users map[discord.UserID]*gateway.PresenceUpdateEvent
	Event chan discord.UserID
	mut   *sync.Mutex
}

// New creates the bot and sets the presence handler
// The session is already opened and the handler is defined here
func New(Token string) *Bot {
	bot := &Bot{
		Pres: &Presence{
			mut:   new(sync.Mutex),
			Users: make(map[discord.UserID]*gateway.PresenceUpdateEvent),
		}}

	// Build session
	var err error
	bot.Sess, err = session.New("Bot " + Token)
	if err != nil {
		log.Fatal().Err(err).Msg("Error aquiring session")
	}

	// Add intents
	bot.Sess.Gateway.AddIntents(gateway.IntentGuildPresences)

	// Set the presence handler
	bot.Sess.AddHandler(bot.PresenceHandler)

	return bot
}

// Run opens the session and starts the bot.
// It blocks until unable to continue
func (b *Bot) Run() error {
	if err := b.Sess.Open(); err != nil {
		return err
	}
	return nil
}

// RunWithEventChan also uses an event channel
// This is convenient if you want your update
// to be real-time for example.
//
// This also means that if the events aren't treated,
// the channel will be stuck and the events won't be cleaned up
func (b *Bot) RunWithEventChan() error {
	b.Pres.Event = make(chan discord.UserID)

	if err := b.Sess.Open(); err != nil {
		return err
	}
	return nil
}

// PresenceHandler handles the discord user presence.
// It uses a map to check if a user's presence should be checked.
// Unless the user is added, the event is discarded.
// The handler just sets the presence of the user inside the map.
func (b *Bot) PresenceHandler(pu *gateway.PresenceUpdateEvent) {
	b.Pres.mut.Lock()
	defer b.Pres.mut.Unlock()

	// Check if user is registered
	if _, ok := b.Pres.Users[pu.User.ID]; !ok {
		return
	}

	// Get user from session,
	// For some reason, the user object is empty on events
	// so I have to do this
	u, err := b.Sess.User(pu.User.ID)
	if err != nil {
		log.Err(err).Msg("Error getting the user")
	}

	// Load user into event
	pu.User = *u
	pu.User.Avatar = fmt.Sprintf("https://cdn.discordapp.com/avatars/%d/%s.png", u.ID, u.Avatar)

	// Sets the presence map
	b.Pres.Users[pu.User.ID] = pu
	if b.Pres.Event != nil {
		b.Pres.Event <- pu.User.ID
	}
}

// Track adds the provided users to the presence handler
func (b *Bot) Track(Users ...discord.UserID) {
	for _, u := range Users {
		b.Pres.Users[u] = nil
	}
}

// Untrack removes the tracking on the provided users
func (b *Bot) Untrack(Users ...discord.UserID) {
	for _, u := range Users {
		delete(b.Pres.Users, u)
	}
}

// LoadUsersFromJSON decodes a json file composed of users to track
// format has to be a basic json array such as `[206794847581896705]`
func (b *Bot) LoadUsersFromJSON(filepath string) error {
	f, err := os.Open(filepath)
	if err != nil {
		return err
	}

	// Unmashal is so cool
	var users []discord.UserID
	err = json.NewDecoder(f).Decode(&users)
	if err != nil {
		return err
	}

	b.Track(users...)
	return nil
}
