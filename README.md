# ReminderDiscordBot
A discord bot that will take events from a config file and send reminders to a discord channel. It can also parse a google sheeet to gather more information about that event.

## Buliding:
`go build main.go -o ReminderDiscordBot`

### Usage:
`./ReminderDiscordBot -t $BOT_TOKEN -s $SHEETS_TOKEN -f config.json`

BOT_TOKEN is the Discord Token that belongs to the bot you have setup and connected to your server. In this case, it is set as an environmental variable, but it could be passed directly to the command line.

SHEETS_TOKEN is the API token for the Google Sheet. You will have to set this up in a Google Cloud account. In this case, it is set as an environmental variable, but it could be passed directly to the command line.

config.json is the json file containing events that the discord channel will recieve reminders about. It needs to follow this format:

```
{
    "events": [
        {
            "name": "Event #1",
            "weekday": "Sunday",
            "discordChannelId": "#####################",
            "tagId": "<@##################>",
            "sheetId": "##############################",
            "sheetRange": "EventSheet!J5:L56",
            "reminderText": "Here is an upcoming event: ",
            "dayToEvent": "0"
        },
        {
            "name": "Event #2",
            "weekday": "Monday",
            "discordChannelId": "#####################",
            "tagId": "<@##################>",
            "sheetId": "##############################",
            "sheetRange": "EventSheet!J5:L56",
            "reminderText": "Here is an upcoming event: ",
            "dayToEvent": "1"
        },
        {
            "name": "Event #3",
            "weekday": "Tuesday",
            "discordChannelId": "#####################",
            "tagId": "<@##################>",
            "sheetId": "##############################",
            "sheetRange": "EventSheet!J5:L56",
            "reminderText": "Here is an upcoming event: ",
            "dayToEvent": "5"
        }
    ]
}
```
#### JSON Parameters
**name**: The name by which the event will be identified.  
**weekday**: The weekday on which the reminder will be sent out.  
**discordChannelId**: The ID of the discord channel where the reminder will be sent.  
**tagId**: The ID of the person/role who will be tagged in the reminder.  
**sheetId** The Google Sheet ID.  
**sheetRange**: The Google Sheet range that will be parsed for information.  
**reminderText**: The text that will be sent prior to the parsed data.  
**dayToEvent**: The number of days between the reminder day and the day of the actual event. This is important to calculate the correct date that will be looked for while parsing the Google Sheet.  

### Running from src
You could also just run it from src using go run:

`go run main.go -t $BOT_TOKEN  -s $SHEETS_TOKEN -f config.json`

## Supported Discord Commands:
** !help ** - Prints a help message
** !listEvents ** - Prints out the names of all the currently configured reminders