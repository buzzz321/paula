package main

import (
	"bufio"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/microcosm-cc/bluemonday"
)

type whatIs struct {
	whoSet        string
	date          string
	whatToExplain string
	explanation   string
}

var (
	whatisDb []whatIs
	mutex    = &sync.Mutex{}
)

func readkey(filename string) string {
	inputFile, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer inputFile.Close()

	scanner := bufio.NewScanner(inputFile)
	scanner.Scan()
	key := scanner.Text()
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading standard input:", err)
	}
	return key
}

func setWhatIs(name string, message string) {
	mutex.Lock()
	defer mutex.Unlock()

	splitted := strings.SplitN(message, " ", 2)
	if len(splitted) < 2 {
		return
	}

	fmt.Println(splitted[0] + " -> " + splitted[1] + "(" + name + ")")
	date := time.Now().String()

	for index, item := range whatisDb {
		if item.whatToExplain == splitted[0] {
			whatisDb[index] = whatIs{name, date, splitted[0], splitted[1]}
			return
		}
	}
	whatisDb = append(whatisDb, whatIs{name, date, splitted[0], splitted[1]})
}

func getWhatIs(what string) (whatIs, int) {
	for index, item := range whatisDb {
		if item.whatToExplain == what {
			return item, index
		}
	}

	return whatIs{"", "", "", ""}, -1
}

func randWhatIs() whatIs {
	mutex.Lock()
	defer mutex.Unlock()

	return whatisDb[rand.Intn(len(whatisDb))]
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	//bail out if not a bot command
	if m.Content[0] != '!' {
		return
	}
	sanitizer := bluemonday.StrictPolicy()

	sanitizeMessage := sanitizer.Sanitize(m.Content)
	sanitizeMessage = strings.ReplaceAll(sanitizeMessage, ";", "")

	splitted := strings.SplitN(sanitizeMessage, " ", 2)

	// well we need to have a command at least
	if len(splitted) < 1 {
		return
	}
	cmd := splitted[0]
	var rest string

	if len(splitted) > 1 {
		rest = splitted[1]
	}

	if cmd == "!randwhatis" {
		whatis := randWhatIs()

		s.ChannelMessageSend(m.ChannelID, whatis.whatToExplain+" -> "+whatis.explanation+"("+whatis.whoSet+")")
	}

	if cmd == "!setwhatis" {
		setWhatIs(m.Author.Username, rest)

		s.ChannelMessageSend(m.ChannelID, "Ping!")
	}

	if cmd == "!whatis" {
		whatis, index := getWhatIs(rest)

		if index == -1 {
			return
		}

		s.ChannelMessageSend(m.ChannelID, whatis.whatToExplain+" -> "+whatis.explanation+"("+whatis.whoSet+")")

	}
}

func readWhatis() {
	inputFile, err := os.Open("whatisdb.txt")
	if err != nil {
		log.Fatal(err)
		return
	}
	defer inputFile.Close()

	scanner := bufio.NewScanner(inputFile)

	for scanner.Scan() {
		line := scanner.Text()

		if err := scanner.Err(); err != nil {
			fmt.Fprintln(os.Stderr, "reading standard input:", err)
		}

		whatisPart := strings.Split(line, ";")
		if len(whatisPart) == 4 {
			fmt.Println(line)
			whatisDb = append(whatisDb, whatIs{whatisPart[0], whatisPart[1], whatisPart[2], whatisPart[3]})
		}
	}

}

func main() {
	rand.Seed(time.Now().Unix())
	discordKey := readkey("../../../discord/paula.key")

	dg, err := discordgo.New("Bot " + discordKey)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	// Register the messageCreate func as a callback for MessageCreate events.
	dg.AddHandler(messageCreate)

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	readWhatis()
	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()

}
