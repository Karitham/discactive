package img

import (
	"bytes"
	"image"
	"io"
	"net/http"
	"os"

	"github.com/rs/zerolog/log"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	// For template file
	_ "embed"

	"image/color"
	"image/draw"
	"image/png"

	"github.com/diamondburned/arikawa/v2/gateway"
)

var (
	// Background is the background image
	Background draw.Image

	// FontFace is the font used to draw text
	FontFace font.Face
)

// This horrible mess is to initialize both
// the font and template.
// TODO: Abstract font and background
func Init(tmpl []byte, ft []byte) {
	GenerateBackground()

	img, err := png.Decode(bytes.NewBuffer(tmpl))
	if err != nil {
		log.Fatal().Err(err).Msg("Couldn't decode the png file, exiting")
	}

	ok := false
	if Background, ok = img.(draw.Image); !ok {
		log.Fatal().Err(err).Msg("Couldn't assert the png to a draw.Image interface, exiting")
	}

	Font, err := opentype.Parse(ft)
	if err != nil {
		log.Fatal().Err(err).Msg("Couldn't parse the font")
	}

	FontFace, err = opentype.NewFace(Font, &opentype.FaceOptions{Size: 13, DPI: 150})
	if err != nil {
		log.Fatal().Err(err).Msg("Couldn't build a usable go font")
	}
}

// UserPresence represents the presence.
// It's used to build the presence image
type UserPresence struct {
	Img   draw.Image
	Color ImageColors
	Font  font.Face
	Pres  Presence
}

// ImageColors is the image background Colors
type ImageColors struct {
	Bg      color.Color
	Profile color.Color
	Banner  color.Color
	Line    color.Color
	Text    color.Color
}

// Presence represents the user's presence
type Presence struct {
	App      string
	Username string
	Avatar   string
	Status   status
}

type status struct {
	color  color.RGBA
	String gateway.Status
}

// Status colors & tags
var (
	// #43B581
	online = status{color.RGBA{R: 0x43, G: 0xB5, B: 0x81, A: 0xFF}, gateway.OnlineStatus}

	// #747E8C
	offline = status{color.RGBA{R: 0x74, G: 0x7E, B: 0x8C, A: 0xFF}, gateway.OfflineStatus}

	// #C84142
	dnd = status{color.RGBA{R: 0xC8, G: 0x41, B: 0x42, A: 0xFF}, gateway.DoNotDisturbStatus}

	// #FAA61A
	idle = status{color.RGBA{R: 0xFA, G: 0xA6, B: 0x1A, A: 0xFF}, gateway.IdleStatus}
)

// New returns a new UserPresence object.
func New(p *gateway.PresenceUpdateEvent) *UserPresence {
	if p == nil {
		return nil
	}

	s := ""
	if len(p.Activities) > 0 {
		s = p.Activities[0].State
	}

	return &UserPresence{
		Img:  Background,
		Font: FontFace,
		Color: ImageColors{
			Bg:      color.RGBA{0x32, 0x35, 0x3B, 0xFF},
			Banner:  color.RGBA{0x2A, 0x2D, 0x33, 0xFF},
			Profile: color.RGBA{0x1B, 0x1E, 0x21, 0xFF},
			Line:    color.RGBA{0x72, 0x89, 0xD9, 0xFF},
			Text:    color.RGBA{0xEE, 0xEE, 0xEE, 0xFF},
		},
		Pres: Presence{
			App:      s,
			Username: p.User.Username + "#" + p.User.Discriminator,
			Avatar:   p.User.Avatar,
			Status:   gatewayStatusToStatus(p.Status),
		},
	}
}

func gatewayStatusToStatus(s gateway.Status) status {
	switch s {
	case online.String:
		return online
	case idle.String:
		return idle
	case dnd.String:
		return dnd
	case offline.String:
		return offline
	}

	return offline
}

// Generate generates the final image
func (u *UserPresence) Generate() (image.Image, error) {
	u.DrawUsername()
	u.DrawProfileImage()
	u.DrawStatus()
	u.DrawApp()

	return u.Img, nil
}

// GenerateBackground generates the background image.
func GenerateBackground() {
	u := &UserPresence{
		Img: image.NewRGBA(image.Rect(0, 0, 600, 140)),
		Color: ImageColors{
			Bg:      color.RGBA{0x32, 0x35, 0x3B, 0xFF},
			Banner:  color.RGBA{0x2A, 0x2D, 0x33, 0xFF},
			Profile: color.RGBA{0x1B, 0x1E, 0x21, 0xFF},
			Line:    color.RGBA{0x72, 0x89, 0xD9, 0xFF},
			Text:    color.RGBA{0xEE, 0xEE, 0xEE, 0xFF},
		},
	}

	// BG
	u.drawRect(
		float64(u.Img.Bounds().Dy()),
		0.25*float64(u.Img.Bounds().Dy()),
		float64(u.Img.Bounds().Dx()),
		0.985*float64(u.Img.Bounds().Dy()),
		u.Color.Bg,
	)

	// Banner
	u.drawRect(
		float64(u.Img.Bounds().Dy()),
		0,
		float64(u.Img.Bounds().Dx()),
		0.25*float64(u.Img.Bounds().Dy()),
		u.Color.Banner,
	)

	// Profile
	u.drawRect(
		0,
		0,
		float64(u.Img.Bounds().Dy()),
		float64(u.Img.Bounds().Dy()),
		u.Color.Profile,
	)

	// Line
	u.drawRect(
		0,
		0.985*float64(u.Img.Bounds().Dy()),
		float64(u.Img.Bounds().Dx()),
		float64(u.Img.Bounds().Dy()),
		u.Color.Line,
	)

	// Template file in assets/template.png
	// Might abstract that and just keep it in memory
	f, err := os.Create("assets/template.png")
	if err != nil {
		log.Fatal().Err(err).Msg("Can't create file template")
	}
	defer f.Close()

	png.Encode(f, u.Img)
}

// drawRect draws a rectangle
func (u *UserPresence) drawRect(x0, y0, x1, y1 float64, c color.Color) {
	for x := x0; x <= x1; x++ {
		for y := y0; y <= y1; y++ {
			u.Img.Set(int(x), int(y), c)
		}
	}
}

// To encodes the file to the writer
func (u *UserPresence) To(w io.Writer) error {
	return png.Encode(w, u.Img)
}

// DrawUsername draws the username of the person
func (u *UserPresence) DrawUsername() {
	var d font.Drawer = font.Drawer{Dst: u.Img, Src: image.NewUniform(u.Color.Text), Dot: fixed.P(160, 26), Face: u.Font}
	d.DrawString(u.Pres.Username)
}

// DrawApp draws the first app of the user
// It's the custom status if set, if not,the app name
func (u *UserPresence) DrawApp() {
	var d font.Drawer = font.Drawer{Dst: u.Img, Src: image.NewUniform(u.Color.Text), Dot: fixed.P(160, 70), Face: u.Font}
	d.DrawString(u.Pres.App)
}

// DrawStatus draws the status of the user
// It's the dot thing on the left
func (u *UserPresence) DrawStatus() {
	draw.DrawMask(
		u.Img,                                 // dst: destination image, written upon
		image.Rect(96, 96, 120, 120),          // r: size of the rectangle to be changed in dst
		image.NewUniform(u.Pres.Status.color), // src: source image
		image.Point{},                         // sp: I don't get what it does but it works
		&circle{image.Point{12, 12}, 12},      // mask: thing that tells the drawer which pixel is transparent or is not
		image.Point{},                         // mp: I don't get what it does but it works
		draw.Over,                             // op: basically just calls draw over every pixel
	)
}

// DrawProfileImage draws the profile image
func (u *UserPresence) DrawProfileImage() {
	res, err := http.Get(u.Pres.Avatar)
	if err != nil {
		log.Err(err).Msg("Error getting avatar")
	}
	defer res.Body.Close()

	m, err := png.Decode(res.Body)
	if err != nil {
		log.Err(err).Msg("Error getting avatar")
	}

	draw.DrawMask(
		u.Img, // dst: destination image, written upon
		image.Rect(6, 6, 6+m.Bounds().Dx(), 6+m.Bounds().Dy()), // r: size of the rectangle to be changed in dst
		m,                                // src: source image
		image.Point{},                    // sp: I don't get what it does but it works
		&circle{image.Point{64, 64}, 64}, // mask: thing that tells the drawer which pixel is transparent or is not
		image.Point{},                    // mp: I don't get what it does but it works
		draw.Over,                        // op: basically just calls draw over every pixel
	)
}

// circle is used as a mask
type circle struct {
	p image.Point
	r int
}

func (c *circle) ColorModel() color.Model {
	return color.AlphaModel
}

func (c *circle) Bounds() image.Rectangle {
	return image.Rect(c.p.X-c.r, c.p.Y-c.r, c.p.X+c.r, c.p.Y+c.r)
}

// At checks if point is inside the circle.
// It's just the pythagorean theorem
// `x^2+y^2 < r^2` with `<` is inside the circle, `==` is on the circle and `>` is outside
//
// It returns an opaque mask inside, and invisible outside
func (c *circle) At(x, y int) color.Color {
	xx, yy, rr := float64(x-c.p.X), float64(y-c.p.Y), float64(c.r)
	if xx*xx+yy*yy < rr*rr {
		return color.Alpha{0xFF}
	}
	return color.Alpha{0}
}
