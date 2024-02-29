package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
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
	Token       string
	SheetsToken string
)

// A Struct for grabbing the Joke from a json object returned from API
type Joke struct {
	Joke string `json: "joke"`
}

// A struct for parsing the sunday lesson info from a google sheet
type SundayLesson struct {
	Date    string
	Topic   string
	Teacher string
}

// A struct for parsing the wednesday activity info from a google sheet
type WednesdayActivity struct {
	Date     string
	Activity string
	Youth    string
}

const JokeURL = "https://v2.jokeapi.dev/joke"

func init() {
	flag.StringVar(&Token, "t", "", "Bot Token")
	flag.StringVar(&SheetsToken, "s", "", "Sheets Token")
	flag.Parse()
}

// A basic help string
var helpMessage string = `Commands:
    !help - Display this message
    !nextActivity - Dispaly info about the next Wednesday night activity
    !nextLesson -  Display info about the next Sunday Lesson
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

	//nextActivity Command
	if m.Content == "!nextActivity" {
		_, err := s.ChannelMessageSend(m.ChannelID, getDateInfo(findNextActivity(), sheetsServiceGlobal))
		if err != nil {
			fmt.Println(err)
		}
	}

	//nextLesson Command
	if m.Content == "!nextLesson" {
		_, err := s.ChannelMessageSend(m.ChannelID, getDateInfo(findNextLesson(), sheetsServiceGlobal))
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

func initReminders(s *discordgo.Session, sheetsService *sheets.Service) {
	// Channel ID for the channel to be posted in
	channelID := "1208196655170715698"

	// Secretary Role ID
	secretaryID := "<@&1212244701697015818>"

	// Wednesday reminder go out 5 hours before
	wedAnticipation := 5 * time.Hour

	// Sunday reminders go out a day before
	sunAnticipation := 24 * time.Hour

	// Find the local time for later calculaiton
	currentTime := time.Now().Local()

	// Find the next lesson and activity
	wedReminderTime := findNextActivity()
	sunReminderTime := findNextLesson()

	// Calculate times between activities and lessons
	wedToSun := sunReminderTime.Sub(wedReminderTime)
	sunToWed := wedReminderTime.Sub(sunReminderTime)

	// Find which one is next
	if wedReminderTime.Before(sunReminderTime) {
		// Calculate time until next activiy and sleep until then
		time.Sleep(wedReminderTime.Sub(currentTime) - wedAnticipation)

		//Send the proper reminder
		sendWedReminder(s, channelID, secretaryID, wedReminderTime, sheetsService)
	} else {
		// Calculate time until next lesson and sleep until then
		time.Sleep(sunReminderTime.Sub(currentTime) - sunAnticipation)

		// Send the proper reminder
		sendSunReminder(s, channelID, secretaryID, sunReminderTime, sheetsService)

	}

	// Create an iterator to use in for loop
	i := 1

	// Continous loop, durations depend on which day it is
	if currentTime.Weekday().String() == "Wednesday" {
		for {
			if i%2 != 0 { // the case where wednesday was the starting day
				time.Sleep(wedToSun)
				sendSunReminder(s, channelID, secretaryID, sunReminderTime, sheetsService)
				i++
			} else {
				time.Sleep(sunToWed)
				sendWedReminder(s, channelID, secretaryID, wedReminderTime, sheetsService)
				i++
			}
		}
	} else if currentTime.Weekday().String() == "Saturday" { // the case where sunday was the starting day
		for {
			if i%2 == 0 {
				time.Sleep(wedToSun)
				sendSunReminder(s, channelID, secretaryID, sunReminderTime, sheetsService)
				i++
			} else {
				time.Sleep(sunToWed)
				sendWedReminder(s, channelID, secretaryID, wedReminderTime, sheetsService)
				i++
			}
		}
	}
}

// Sends the sunday reminder message
func sendSunReminder(s *discordgo.Session, chanID string, secRoleID string, dateToFind time.Time, sheetsService *sheets.Service) {
	// Spreadsheet Info
	spreadsheetID := "10vomq2rhxO-pS664uD9lhLol1nJpfmbxYu8hpfOLJi8"
	// This points to the sunday cell range
	cellRange := "Priests!J5:L56"
	// Parse the range to find the right info
	parsedText := parseSpreadsheet(dateToFind, sheetsService, spreadsheetID, cellRange)
	// Create the final string
	sundayReminderText := secRoleID + " Please send a reminder about the lesson for tomorrow if there is one:   " + parsedText
	//Send the string to the channel
	_, err := s.ChannelMessageSend(chanID, sundayReminderText)
	if err != nil {
		fmt.Println(err)
	}
}

// Sends the Wednesday Reminder message
func sendWedReminder(s *discordgo.Session, chanID string, secRoleID string, dateToFind time.Time, sheetsService *sheets.Service) error {
	// Spreadsheet Info
	spreadsheetID := "10vomq2rhxO-pS664uD9lhLol1nJpfmbxYu8hpfOLJi8"
	// This points to the sunday cell range
	cellRange := "Priests!C5:E56"
	// Parse the range to find the right info
	parsedText := parseSpreadsheet(dateToFind, sheetsService, spreadsheetID, cellRange)
	// Create the final string
	wednesdayReminderText := secRoleID + " Please send a reminder about the activity if there is one:   " + parsedText
	//Send the string to the channel
	_, err := s.ChannelMessageSend(chanID, wednesdayReminderText)
	if err != nil {
		fmt.Println(err)
	}
	return nil
}

func getDateInfo(date time.Time, sheetsService *sheets.Service) string {
	// Spreadsheet Info
	spreadsheetID := "10vomq2rhxO-pS664uD9lhLol1nJpfmbxYu8hpfOLJi8"
	//Empty Cell range to fill
	cellRange := ""
	// Choose cellRange to parse based on weekday
	if date.Weekday().String() == "Sunday" {
		cellRange = "Priests!J5:L56"
	} else {
		cellRange = "Priests!C5:E56"
	}
	// Parse the range to find the right info
	return parseSpreadsheet(date, sheetsService, spreadsheetID, cellRange)
}

// Function to find the date of the next Activity
func findNextActivity() time.Time {
	//Find the next wednesday
	//Get current day
	now := time.Now().Local()
	activityTime := time.Date(now.Year(), now.Month(), now.Day(), 19, 0, 0, 0, now.Location())
	//Find current weekday
	weekday := int(now.Weekday())
	//Default to current day, if it is not a wednesday we will change it
	nextDate := activityTime
	//Days before wednesday
	if weekday < 3 {
		nextDate = activityTime.AddDate(0, 0, 3-int(weekday))
	} else if weekday > 3 {
		nextDate = activityTime.AddDate(0, 0, 10-int(weekday))
	}

	return nextDate
}

// Function to find the next lesson information and format it as string
func findNextLesson() time.Time {
	//Find the next Sunday
	//Get current day
	now := time.Now().Local()
	activityTime := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, now.Location())
	//Find current weekday
	weekday := int(now.Weekday())
	//Default to current day, if it is not a Sunday we will change it
	nextDate := activityTime
	//Days before Sunday
	if weekday < 7 {
		nextDate = activityTime.AddDate(0, 0, 7-int(weekday))
	}
	return nextDate
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
