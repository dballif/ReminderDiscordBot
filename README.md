# Go-Discord-Bot
A discord bot that will grab data from a google sheet and send updates to a discord server

## Buliding:
`go build main.go -o DiscordBot`

### Usage:
`./DiscordBot -t $BOT_TOKEN`

BOT_TOKEN is the Discord Token that belongs to the bot you have setup and connected to your server. In this case, it is set as an environmental variable, but it could be passed directly to the command line.

### Running from src
You could also just run it from src using go run:

`go run main.go -t $BOT_TOKEN`


## Supported Discord Commands:
** !help ** - Prints a help message
** !nextActivity ** - Checks the spreadsheet for the next wendesday activity
** !nextLesson ** - Checks the spreadhsheet for info on the next Sunday Lesson
** !joke ** - Grabs a joke from a joke API