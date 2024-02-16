package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
)

// Variables used for command line parameters
var (
	Token string
)

const JokeURL = "https://v2.jokeapi.dev/joke"

func init() {
	flag.StringVar(&Token, "t", "", "Bot Token")
	flag.Parse()
}

var helpMessage string = `Commands:
    !help - Display this message
    !nextActivity - Dispaly info about the next Wednesday night activity
    !nextLesson -  Display info about the next Sunday Lesson
`

// Main Function
func main() {

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + Token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	// Register the messageCreate func as a callback for MessageCreate events.
	dg.AddHandler(messageCreate)

	// In this example, we only care about receiving message events.
	dg.Identify.Intents = discordgo.IntentsGuildMessages

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	dg.Close()
}

// A Struct for grabbing the Joke from a json object returned from API
type Joke struct {
	Joke string `json: "joke"`
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the authenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		return
	}

	//Help Command
	if m.Content == "!help" {
		_, err := s.ChannelMessageSend(m.ChannelID, helpMessage)
		if err != nil {
			fmt.Println(err)
		}
	}

	//nextActivity Command
	if m.Content == "!nextActivity" {
		_, err := s.ChannelMessageSend(m.ChannelID, findNextActivity())
		if err != nil {
			fmt.Println(err)
		}
	}

	//nextLesson Command
	if m.Content == "!nextLesson" {
		fmt.Println("next lesson called")
		_, err := s.ChannelMessageSend(m.ChannelID, findNextLesson())
		if err != nil {
			fmt.Println(err)
		}
	}

	// !joke Command
	if m.Content == "!joke" {
		// Try to send the joke
		_, err := s.ChannelMessageSend(m.ChannelID, getJoke())
		if err != nil {
			fmt.Println(err)
		}
	}
}

// Function to find the next activity information and format it as a string
func findNextActivity() string {
	//String to be filled
	var outputString = ""
	//Find the next wednesday
	//Get current day
	now := time.Now()
	//Find current weekday
	weekday := int(time.Now().Weekday())
	//Default to current day, if it is not a wednesday we will change it
	nextDate := time.Now()
	//Days before wednesday
	if weekday < 3 {
		nextDate = now.AddDate(0, 0, 3-int(weekday))
	} else if weekday > 3 {
		nextDate = now.AddDate(0, 0, 10-int(weekday))
	}

	//Find the corresponding entry in the sheet
	//Can use now.compare to check if it is the the same date?
	//May have to reformat the date coming from sheets

	//Dummy holder
	outputString = nextDate.Weekday().String()

	return outputString
}

// Function to find the next lesson information and format it as string
func findNextLesson() string {
	var outputString = ""
	return outputString
}

func getJoke() string {
	//Call the Joke API and retrieve our cute Dr Who Gopher
	response, err := http.Get(JokeURL + "/Any?blacklistFlags=nsfw,religious,racist,sexist&type=single")
	if err != nil {
		fmt.Println(err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		fmt.Println(err)
	}

	var jokeData Joke
	err = json.Unmarshal(body, &jokeData)
	if err != nil {
		fmt.Println(err)
	}

	return jokeData.Joke
}
