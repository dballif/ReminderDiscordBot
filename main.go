package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

// Variables used for command line parameters
var (
	Token        string
	SheetsToken  string
	ConfigFile   string
	ReminderHour int
)

// Individual Events
type ReminderEvent struct {
	Name             string
	Weekday          string
	DiscordChannelId string
	TagId            string
	SheetId          string
	SheetRange       string
	ReminderText     string
	DayToEvent       string
}

type Events struct {
	Events []ReminderEvent `json:"events"`
}

func init() {
	flag.StringVar(&Token, "t", "", "Bot Token")
	flag.StringVar(&SheetsToken, "s", "", "Sheets Token")
	flag.StringVar(&ConfigFile, "f", "config.json", "Json Config File")
	flag.IntVar(&ReminderHour, "r", 12, "Reminder Time (Hour)")
	flag.Parse()
}

// A basic help string
var helpMessage string = `Commands:
    !help - Display this message
	!listEvents - list all the currently configured event reminders
`

// The global sheets Service ot make it easy to access - This is really only needed by Handler since we can't modify what is given to them
var sheetsServiceGlobal *sheets.Service

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

	// Prepare Google Sheets
	// API Context
	ctx := context.Background()

	// Create the google sheets service
	service, err := sheets.NewService(ctx, option.WithAPIKey(SheetsToken))
	if err != nil {
		fmt.Println(err)
	}

	// Fill the global service
	sheetsServiceGlobal = service

	// Kickoff the reminder thread
	go initReminders(dg, service)

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

func parseSpreadsheet(dateToFind time.Time, service *sheets.Service, ID string, cellRange string) string {
	foundEntry := ""
	getActivityResponse, err := service.Spreadsheets.Values.Get(ID, cellRange).Do()
	if err != nil {
		fmt.Println("Get response")
		fmt.Println(err)
	}

	// Convert the date to a string in the right format - MM/DD/YYYY
	dateToFindYear, dateToFindMonth, dateToFindDay := dateToFind.Date()
	convertedDate := strconv.Itoa(int(dateToFindMonth)) + "/" + strconv.Itoa(dateToFindDay) + "/" + strconv.Itoa(dateToFindYear)

	// Now I need to parse each response into a struct - Date,Activity,User
	for _, s := range getActivityResponse.Values {
		// Check if date matches the one we are parsing for
		if s[0].(string) == convertedDate {
			// Convert to String
			// Get the Date
			if len(s) >= 1 {
				foundEntry += s[0].(string) + " | "
			} else {
				foundEntry += "TBD | "
			}
			// Get the Activity/Lesson
			if len(s) >= 2 {
				foundEntry += s[1].(string) + " | "
			} else {
				foundEntry += "TBD | "
			}
			// Get the person in charge
			if len(s) >= 3 {
				foundEntry += s[2].(string) + " | "
			} else {
				foundEntry += "TBD"
			}
		}
	}
	return foundEntry
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

	if m.Content == "!listEvents" {
		// Parse the JSON for information
		eventArray := parseJsonFile(ConfigFile)
		// Create a variable to hold the string that will be sent
		var eventListStr string
		eventListStr += "----------Configured Events----------\n"
		// Iterate over the entire array
		for _, event := range eventArray.Events {
			eventListStr += "Event Name: " + event.Name
			eventListStr += "\n"
		}
		// Send the message
		_, err := s.ChannelMessageSend(m.ChannelID, eventListStr)
		if err != nil {
			fmt.Println(err)
		}
	}
}

func parseJsonFile(configFile string) Events {
	var eventsData Events

	// Check if file exists
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		fmt.Println("File does not exist:", configFile)
	}

	file, err := os.Open(configFile)
	if err != nil {
		fmt.Println("Open error")
	}

	jsonData, err := io.ReadAll(file)
	if err != nil {
		fmt.Println("Read all error")
	}

	err = json.Unmarshal(jsonData, &eventsData)
	if err != nil {
		fmt.Println("unmarshal error")
	}

	return eventsData
}

func initReminders(s *discordgo.Session, sheetsService *sheets.Service) {
	//Parse the config file json to find the events to start looking for
	eventArray := parseJsonFile(ConfigFile)

	// Calculate time to daily check
	// FIXME: will eventually be configurable
	remindTime := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), ReminderHour, 0, 0, 0, time.Local)

	// If the target time has already passed today, move it to the next day
	if time.Now().After(remindTime) {
		remindTime = remindTime.Add(24 * time.Hour)
	}

	timeUntilRemind := time.Until(remindTime)
	fmt.Println("Time until next remind: " + timeUntilRemind.String())
	time.Sleep(timeUntilRemind)

	sleepTime := 24 * time.Hour //FIXME: Should probably be configurable

	//Now just loop every 24 hours to check at the same time everyday
	i := 0
	daysToRemind := 365
	for i < daysToRemind {
		i++
		for _, event := range eventArray.Events {
			// Run through events in config file and check to see if they need a reminder
			if event.Weekday == time.Now().Weekday().String() {
				fmt.Println("Sending reminder for: " + event.Name)
				sendReminder(s, event, sheetsService)
			}
		}
		time.Sleep(sleepTime)
	}
}

// Sends the sunday reminder message
func sendReminder(s *discordgo.Session, event ReminderEvent, sheetsService *sheets.Service) {
	//FIXME: Adjust the parsedReminderText to tag the person involved

	// Convert string form json into int
	dayUntilremind, errs := strconv.Atoi(event.DayToEvent)
	if errs != nil {
		fmt.Println("Conversion to Int error")
	}
	// Time until actual event (in hours)
	dateToFind := time.Duration(24 * dayUntilremind)
	dateToFind = dateToFind * time.Hour

	// Parse the range to find the right info
	parsedText := parseSpreadsheet(time.Now().Add(dateToFind), sheetsService, event.SheetId, event.SheetRange)
	parsedReminderText := event.TagId + event.ReminderText + " " + parsedText
	//Send the string to the channel
	_, err := s.ChannelMessageSend(event.DiscordChannelId, parsedReminderText)
	if err != nil {
		fmt.Println(err)
	}
}
