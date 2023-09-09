package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/kksmiles/asotaw-bot/dgvoice"
)

const PREFIX = "!gosing"
const VIEW_QUEUE_COMMAND = "!gosing-queue"
const PAUSE_COMMAND = "!gosing-pause"
const RESUME_COMMAND = "!gosing-resume"
const SKIP_COMMAND = "!gosing-skip"
const LEAVE_COMMAND = "!gosing-leave"
const LOOP_ONE_COMMAND = "!gosing-loop-one"
const LOOP_ALL_COMMAND = "!gosing-loop-all"
const LOOP_COMMAND = "!gosing-loop"
const FOLDER = "audio"
const YOUTUBE_SEARCH_ENDPOINT = "https://www.googleapis.com/youtube/v3/search"

type fileQueue struct {
	fileName    string
	videoTitle  string
	readyToPlay bool
	playTime    int64
	startTime   int64
	dgv         *discordgo.VoiceConnection
	stopChannel chan bool
}

type YouTubeResponse struct {
	Items []struct {
		ID struct {
			VideoID string `json:"videoId"`
		} `json:"id"`
		Snippet struct {
			Title string `json:"title"`
		}
	} `json:"items"`
}

var queue map[string][]fileQueue = make(map[string][]fileQueue)
var runningGuilds map[string]bool = make(map[string]bool)
var pausedGuilds map[string]bool = make(map[string]bool)
var loopGuilds map[string]string = make(map[string]string)
var newGuildDetected chan bool = make(chan bool)

func main() {
	// Create a new Discord session
	DISCORD_TOKEN := os.Getenv("DISCORD_TOKEN")
	session, err := discordgo.New("Bot " + DISCORD_TOKEN)
	if err != nil {
		fmt.Println("Error creating Discord session:", err)
		return
	}

	// Register an event handler
	session.AddHandler(onMessageCreate)
	session.Identify.Intents = discordgo.IntentsAllWithoutPrivileged

	// Open a connection to Discord
	err = session.Open()
	if err != nil {
		fmt.Println("Error opening connection to Discord:", err)
		return
	}

	//  GoRoutine : Process queue for each guild
	go func() {
		for {
			if <-newGuildDetected {
				fmt.Print("New guild detected. ")
				for guildId := range queue {
					if !runningGuilds[guildId] {
						runningGuilds[guildId] = true
						go runForGuild(session, guildId)
					}
				}
			}
		}
	}()

	// Keep the bot running
	fmt.Println("Bot is now running. Press Ctrl+C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}

func runForGuild(session *discordgo.Session, guildId string) {
	runningGuilds[guildId] = true

	for {
		time.Sleep(1 * time.Second)
		fmt.Println("Guild ID :", guildId)

		if len(queue[guildId]) == 0 {
			fmt.Println("Queue is empty. Stopping guild.")
			runningGuilds[guildId] = false
			botLeaveVoiceChannel(session, guildId)
			return
		}

		if pausedGuilds[guildId] {
			continue
		}

		if !queue[guildId][0].readyToPlay {
			fmt.Println("File not ready to play:", queue[guildId][0].fileName)
			time.Sleep(5 * time.Second)
			continue
		}

		playFirstTrackOfGuild(session, guildId)

		if !pausedGuilds[guildId] {
			if loopGuilds[guildId] == "one" {
				queue[guildId][0].playTime = 0
			} else {
				if loopGuilds[guildId] == "all" {
					queue[guildId] = append(queue[guildId], queue[guildId][0])
				}
				if len(queue[guildId]) > 1 {
					queue[guildId] = queue[guildId][1:]
				} else {
					queue[guildId] = []fileQueue{}
				}
			}

		}
	}
}

func getYoutubeVideo(query string) (YouTubeResponse, error) {
	YOUTUBE_TOKEN := os.Getenv("YOUTUBE_TOKEN")
	url := fmt.Sprintf("%s?key=%s&part=snippet&q=%s&maxResults=1&type=video", YOUTUBE_SEARCH_ENDPOINT, YOUTUBE_TOKEN, url.QueryEscape(query))

	response, err := http.Get(url)
	if err != nil {
		return YouTubeResponse{}, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return YouTubeResponse{}, fmt.Errorf("response status code was %d", response.StatusCode)
	}

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return YouTubeResponse{}, err
	}

	var youtubeResponse YouTubeResponse
	if err := json.Unmarshal(responseBody, &youtubeResponse); err != nil {
		return YouTubeResponse{}, err
	}

	return youtubeResponse, nil
}

func downloadYoutubeVideo(videoUrl string, fileName string) error {
	// Check if file already exists
	if _, err := os.Stat(fmt.Sprintf("%s/%s", FOLDER, fileName)); err == nil {
		fmt.Println("File already exists:", fileName)
		return nil
	}

	output := fmt.Sprintf("%s/%s", FOLDER, fileName)
	command := exec.Command("yt-dlp", "-x", "--audio-format", "mp3", "--audio-quality", "0", "-o", output, videoUrl)

	stdout, err := command.StdoutPipe()
	if err != nil {
		fmt.Println("Error creating stdout pipe:", err)
		return err
	}

	stderr, err := command.StderrPipe()
	if err != nil {
		fmt.Println("Error creating stderr pipe:", err)
		return err
	}

	fmt.Print(command.String())
	err = command.Start()
	if err != nil {
		fmt.Println("Error:", err)
		return err
	}

	// Create a goroutine to print the command's stdout
	go func() {
		io.Copy(os.Stdout, stdout)
	}()

	// Create a goroutine to print the command's stderr
	go func() {
		io.Copy(os.Stderr, stderr)
	}()

	err = command.Wait()
	if err != nil {
		fmt.Println("Error:", err)
		return err
	}

	return nil
}

func onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore messages from the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}
	fmt.Println("Message received: " + m.Content)

	// Check if the message starts with the PREFIX
	args := strings.Split(m.Content, " ")

	if !strings.HasPrefix(args[0], PREFIX) {
		return
	}

	switch args[0] {
	case LEAVE_COMMAND:
		botLeaveVoiceChannel(s, m.GuildID)
		return
	case VIEW_QUEUE_COMMAND:
		viewQueue(s, m)
		return
	case PAUSE_COMMAND:
		pauseTrack(s, m)
		return
	case RESUME_COMMAND:
		resumeTrack(s, m)
		return
	case SKIP_COMMAND:
		skipTrack(s, m)
		return
	case LOOP_ONE_COMMAND:
		loopGuilds[m.GuildID] = "one"
		s.ChannelMessageSend(m.ChannelID, "Looping the current track.")
		return
	case LOOP_ALL_COMMAND:
		loopGuilds[m.GuildID] = "all"
		s.ChannelMessageSend(m.ChannelID, "Looping the entire queue.")
		return
	case LOOP_COMMAND:
		if loopGuilds[m.GuildID] == "one" {
			loopGuilds[m.GuildID] = "all"
			s.ChannelMessageSend(m.ChannelID, "Looping the entire queue.")
		} else if loopGuilds[m.GuildID] == "all" {
			loopGuilds[m.GuildID] = "none"
			s.ChannelMessageSend(m.ChannelID, "Looping disabled.")
		} else {
			loopGuilds[m.GuildID] = "one"
			s.ChannelMessageSend(m.ChannelID, "Looping the current track.")
		}
	}

	// Check if the user is in a voice channel
	voiceState, err := s.State.VoiceState(m.GuildID, m.Author.ID)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "Please join a voice channel first.")
		return
	}

	// Check if the bot is already in a voice channel
	botVoiceState, err := s.State.VoiceState(m.GuildID, s.State.User.ID)
	if err == nil {
		if botVoiceState.ChannelID != "" && botVoiceState.ChannelID != voiceState.ChannelID {
			s.ChannelMessageSend(m.ChannelID, "I'm already in a voice channel.")
			return
		}
	}

	// Join the voice channel
	dgv, err := s.ChannelVoiceJoin(m.GuildID, voiceState.ChannelID, false, true)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "Error joining voice channel.")
		return
	}

	// Search on Youtube
	query := strings.Join(args[1:], " ")
	fmt.Println("Search Query: " + query)

	videoInstance, err := getYoutubeVideo(query)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "Error searching on Youtube.")
		return
	}
	if len(videoInstance.Items) == 0 {
		s.ChannelMessageSend(m.ChannelID, "No results found.")
		return
	}

	// Add the file to the queue
	videoID := videoInstance.Items[0].ID.VideoID
	videoTitle := videoInstance.Items[0].Snippet.Title
	videoURL := fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID)
	var fileName = videoID + ".mp3"

	s.ChannelMessageSend(m.ChannelID, "Added "+videoTitle+" to the queue.")
	currentIndex := len(queue[m.GuildID])
	stopChannel := make(chan bool)
	queue[m.GuildID] = append(queue[m.GuildID], fileQueue{fileName, videoTitle, false, 0, 0, dgv, stopChannel})

	if !runningGuilds[m.GuildID] {
		newGuildDetected <- true
	}

	// Download the file
	downloadYoutubeVideo(videoURL, fileName)

	// Mark the file as ready to play
	queue[m.GuildID][currentIndex].readyToPlay = true

	fmt.Println(fileName + " is now ready to play.")
}

func playFirstTrackOfGuild(s *discordgo.Session, guildId string) {
	fq := &queue[guildId][0]

	fq.startTime = time.Now().Unix()
	fmt.Println("PlayTime", fq.playTime)
	s.ChannelMessageSend(fq.dgv.ChannelID, "Now playing "+fq.videoTitle)

	dgvoice.PlayAudioFile(fq.dgv, fmt.Sprintf("%s/%s", FOLDER, fq.fileName), strconv.FormatInt(fq.playTime, 10), fq.stopChannel)
}

func pauseTrack(s *discordgo.Session, m *discordgo.MessageCreate) {
	diff := time.Now().Unix() - queue[m.GuildID][0].startTime
	queue[m.GuildID][0].playTime += diff

	queue[m.GuildID][0].stopChannel <- true
	pausedGuilds[m.GuildID] = true
	s.ChannelMessageSend(m.ChannelID, "Paused the current track.")
}

func resumeTrack(s *discordgo.Session, m *discordgo.MessageCreate) {
	pausedGuilds[m.GuildID] = false
	s.ChannelMessageSend(m.ChannelID, "Resumed the current track.")
}

func skipTrack(s *discordgo.Session, m *discordgo.MessageCreate) {
	queue[m.GuildID][0].stopChannel <- true
	s.ChannelMessageSend(m.ChannelID, "Skipped the current track.")
}

func viewQueue(s *discordgo.Session, m *discordgo.MessageCreate) {
	var message string
	for i, fq := range queue[m.GuildID] {
		if i == 0 {
			message += fmt.Sprintf("(Now Playing) %s\n Up Next : \n", fq.videoTitle)
		} else {
			message += fmt.Sprintf("%d. %s \n", i, fq.videoTitle)
		}
	}
	s.ChannelMessageSend(m.ChannelID, message)
}

func clearQueue(guildId string) {
	if len(queue[guildId]) > 0 {
		queue[guildId][0].stopChannel <- true
	}
	queue[guildId] = []fileQueue{}
	pausedGuilds[guildId] = false
	loopGuilds[guildId] = "none"
}

func botLeaveVoiceChannel(s *discordgo.Session, guildId string) {
	clearQueue(guildId)
	s.ChannelVoiceJoinManual(guildId, "", false, true)
}
