package main

import (
	"bufio"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
)

type whatIs struct {
	who   string
	what  string
	entry string
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

func setWhatis(name string, message string) {
	mutex.Lock()
	defer mutex.Unlock()

	splitted := strings.SplitN(message, " ", 2)
	if len(splitted) < 2 {
		return
	}

	fmt.Println(splitted[0] + " -> " + splitted[1] + "(" + name + ")")
	whatisDb = append(whatisDb, whatIs{name, splitted[0], splitted[1]})

}

func randWhatis() whatIs {
	mutex.Lock()
	defer mutex.Unlock()
	size := len(whatisDb)

	fmt.Println("size:" + strconv.Itoa(size))
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

	splitted := strings.SplitN(m.Content, " ", 2)

	// well we need to have a command at least
	if len(splitted) < 1 {
		return
	}
	cmd := splitted[0]
	var rest string

	if len(splitted) > 1 {
		rest = splitted[1]
	}

	// If the message is "ping" reply with "Pong!"
	if cmd == "!randwhatis" {
		whatis := randWhatis()

		s.ChannelMessageSend(m.ChannelID, whatis.what+" -> "+whatis.entry+"("+whatis.who+")")
	}

	// If the message is "pong" reply with "Ping!"
	if cmd == "!setwhatis" {
		setWhatis(m.Author.Username, rest)

		s.ChannelMessageSend(m.ChannelID, "Ping!")
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

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()

}
