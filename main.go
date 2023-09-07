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
	"time"

	"github.com/bwmarrin/dgvoice"
	"github.com/bwmarrin/discordgo"
)

const PREFIX = "!gosing"
const QUEUE_COMMAND = "!gosing-queue"
const LEAVE_COMMAND = "!gosing-leave"
const FOLDER = "audio"
const YOUTUBE_SEARCH_ENDPOINT = "https://www.googleapis.com/youtube/v3/search"

type fileQueue struct {
	fileName   string
	videoTitle string
	dgv        *discordgo.VoiceConnection
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

	//  GoRoutine : Process queue
	go func() {
		for {
			fmt.Println("Processing queue")
			for guildId, fqs := range queue {
				fmt.Println("Guild ID:", guildId)
				if len(fqs) == 0 {
					continue
				}

				for _, fq := range fqs {
					fmt.Println("Playing audio file:", fq.fileName)
					playAudio(session, fq.dgv, fq.fileName, fq.videoTitle)

					// Remove the file from the queue
					queue[guildId] = queue[guildId][1:]
				}
			}
			time.Sleep(5 * time.Second)
		}
	}()

	// Keep the bot running
	fmt.Println("Bot is now running. Press Ctrl+C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
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
	if args[0] == LEAVE_COMMAND {
		// Clear queue
		queue[m.GuildID] = nil

		// Leave discord voice channel
		s.ChannelMessageSend(m.ChannelID, "Leaving voice channel.")
		s.ChannelVoiceJoinManual(m.GuildID, "", false, true)
		return
	}
	if args[0] == QUEUE_COMMAND {
		viewQueue(s, m)
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
	downloadYoutubeVideo(videoURL, fileName)
	queue[m.GuildID] = append(queue[m.GuildID], fileQueue{fileName, videoTitle, dgv})
	fmt.Println("Added to queue:", fileName)
}

func playAudio(s *discordgo.Session, dgv *discordgo.VoiceConnection, fileName string, videoTitle string) {
	fmt.Println("Play Audio File:", fileName)
	s.ChannelMessageSend(dgv.ChannelID, "Now playing "+videoTitle)
	dgvoice.PlayAudioFile(dgv, fmt.Sprintf("%s/%s", FOLDER, fileName), make(chan bool))
}

func viewQueue(s *discordgo.Session, m *discordgo.MessageCreate) {
	var message string
	for i, fq := range queue[m.GuildID] {
		message += fmt.Sprintf("%d. ", i+1)
		message += fq.videoTitle + "\n"
	}
	s.ChannelMessageSend(m.ChannelID, "Up Next: \n"+message)
}
