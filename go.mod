module github.com/kksmiles/asotaw-bot

go 1.21.0

require (
	github.com/bwmarrin/discordgo v0.27.1
	github.com/kksmiles/asotaw-bot/dgvoice v0.0.0-00010101000000-000000000000
)

replace github.com/kksmiles/asotaw-bot/dgvoice => ./dgvoice

require (
	github.com/gorilla/websocket v1.4.2 // indirect
	golang.org/x/crypto v0.12.0 // indirect
	golang.org/x/sys v0.11.0 // indirect
	layeh.com/gopus v0.0.0-20210501142526-1ee02d434e32 // indirect
)
