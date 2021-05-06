package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

var rolesMap map[string]string
var membersMap map[string]string

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	token := os.Getenv("TOKEN")
	if token == "" {
		fmt.Println("No token provided. Please provide a .env file or run: TOKEN={TOKEN} ./RoleBot")
		return
	}

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println("Error creating Discord session: ", err)
		return
	}

	// Register guildCreate as a callback for the guildCreate events.
	dg.AddHandler(guildCreate)

	// We need information about guilds (which includes their channels),
	// messages and voice states.
	dg.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMembers

	// Open the websocket and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("Error opening Discord session: ", err)
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("RoleBot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()
}

// This function will be called (due to AddHandler above) every time a new
// guild is joined.
func guildCreate(s *discordgo.Session, event *discordgo.GuildCreate) {

	rolesMap = make(map[string]string)
	membersMap = make(map[string]string)

	if event.Guild.Unavailable {
		return
	}

	for _, role := range event.Guild.Roles {
		fmt.Printf("%+v\n", role)
		rolesMap[role.Name] = role.ID
	}

	members, _ := s.GuildMembers(event.Guild.ID, "", 1000)
	for _, m := range members {
		membersMap[m.User.Username+"#"+m.User.Discriminator] = m.User.ID
	}

	grantRoles(s, event)
}

// grantRoles reads the testers.csv and assigns all people the proper roles
func grantRoles(s *discordgo.Session, event *discordgo.GuildCreate) {
	// Open the file
	csvfile, err := os.Open("testers.csv")
	if err != nil {
		log.Fatalln("Couldn't open the csv file", err)
	}

	// Parse the file
	r := csv.NewReader(csvfile)
	//r := csv.NewReader(bufio.NewReader(csvfile))

	// Iterate through the records
	for {
		// Read each record from csv
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		if _, ok := membersMap[record[0]]; ok {
			continue // Member is not in this server
		}

		if _, ok := rolesMap[record[1]]; ok {
			fmt.Printf("ERROR: Unknown role %s\n", record[1])
			continue // Role doesn't exist
		}

		s.GuildMemberRoleAdd(event.Guild.ID, membersMap[record[0]], rolesMap[record[1]])
		fmt.Printf("Added %s to Role %s\n", record[0], record[1])
	}
}
