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
	"strings"
	"syscall"

	"github.com/bwmarrin/dgvoice"
	"github.com/bwmarrin/discordgo"
)

const PREFIX = "!gosing"
const LEAVE_COMMAND = "!gosing-leave"

const FOLDER = "audio"
const YOUTUBE_SEARCH_ENDPOINT = "https://www.googleapis.com/youtube/v3/search"

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

	// Keep the bot running
	fmt.Println("Bot is now running. Press Ctrl+C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}

type YouTubeResponse struct {
	Items []struct {
		ID struct {
			VideoID string `json:"videoId"`
		} `json:"id"`
	} `json:"items"`
}

func getYoutubeVideoID(query string) (string, error) {
	YOUTUBE_TOKEN := os.Getenv("YOUTUBE_TOKEN")
	url := fmt.Sprintf("%s?key=%s&part=snippet&q=%s&maxResults=1&type=video", YOUTUBE_SEARCH_ENDPOINT, YOUTUBE_TOKEN, url.QueryEscape(query))

	// Send a GET request to the YouTube API
	response, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	// Check the response status code
	if response.StatusCode != http.StatusOK {
		return "", err
	}

	// Read the response body
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	// Parse the JSON response
	var youtubeResponse YouTubeResponse
	if err := json.Unmarshal(responseBody, &youtubeResponse); err != nil {
		return "", err
	}

	// Print the response as a string
	return youtubeResponse.Items[0].ID.VideoID, nil
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
	if args[0] == LEAVE_COMMAND {
		// Leave discord voice channel
		s.ChannelMessageSend(m.ChannelID, "Leaving voice channel.")
		s.ChannelVoiceJoinManual(m.GuildID, "", false, true)
		return
	}
	if args[0] != PREFIX {
		return
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
	fmt.Println("Query: " + query)

	videoID, err := getYoutubeVideoID(query)
	videoURL := fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID)
	var fileName = videoID + ".mp3"

	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "Error searching on Youtube.")
		return
	}
	s.ChannelMessageSend(m.ChannelID, "Added "+videoURL+" to the queue. Please wait while I remember how to sing.")
	downloadYoutubeVideo(videoURL, fileName)

	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "Error searching on Youtube.")
		return
	}

	fmt.Println("PlayAudioFile:", fileName)
	s.ChannelMessageSend(m.ChannelID, "Now playing "+videoURL)
	dgvoice.PlayAudioFile(dgv, fmt.Sprintf("%s/%s", FOLDER, fileName), make(chan bool))

	s.ChannelMessageSend(m.ChannelID, "Leaving voice channel.")
	s.ChannelVoiceJoinManual(m.GuildID, "", false, true)
}
