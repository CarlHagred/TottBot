package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/Andreychik32/ytdl"
	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

const (
	prefix = "tott"
)

func main() {
	godotenv.Load()
	token := os.Getenv("BOT_TOKEN")
	discord, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatal(err)
	}

	discord.AddHandler(inputHandler)

	discord.Identify.Intents = discordgo.IntentsAllWithoutPrivileged

	err = discord.Open()
	if err != nil {
		log.Fatal("Error opening connection:", err)
	}

	defer discord.Close()

	fmt.Println("Bot is now online. Press CTRL+C to exit.")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
	_ = discord.Close()
}

func inputHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	args := strings.Split(m.Content, " ")

	if args[0] != prefix {
		return
	}

	if args[1] == "August" {
		s.ChannelMessageSend(m.ChannelID, "Den killen är kort!")
	}

	if args[1] == "play" {
		voiceChannelID, err := getVoiceChannelID(s, m.GuildID, m.Author.ID)
		if err != nil {
			fmt.Println("Error finding user's voice channel:", err)
			return
		}

		if voiceChannelID == "" {
			fmt.Println("User is not in a voice channel.")
			return
		}

		audioURL, err := extractYouTubeAudioURL("https://www.youtube.com/watch?v=y6120QOlsfU")
		if err != nil {
			fmt.Println("Error extracting YouTube audio URL:", err)
			return
		}

		vc, err := s.ChannelVoiceJoin(m.GuildID, voiceChannelID, false, false)
		if err != nil {
			fmt.Println("Error joining voice channel:", err)
			return
		}
		defer vc.Disconnect()

		err = downloadAndPlayAudio(vc, audioURL)
		if err != nil {
			fmt.Println("Error playing audio:", err)
			return
		}
	}
}

func getVoiceChannelID(s *discordgo.Session, guildID, userID string) (string, error) {
	guild, err := s.Guild(guildID)
	if err != nil {
		return "", err
	}

	for _, vs := range guild.VoiceStates {
		if vs.UserID == userID {
			return vs.ChannelID, nil
		}
	}

	return "", nil
}

func downloadAndPlayAudio(vc *discordgo.VoiceConnection, audioURL string) error {
	resp, err := http.Get(audioURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	done := make(chan bool)
	errChan := make(chan error)

	go func() {
		err := vc.Speaking(true)
		if err != nil {
			errChan <- err
			return
		}

		_, err = io.Copy(vc, resp.Body)
		if err != nil {
			errChan <- err
			return
		}

		err = vc.Speaking(false)
		if err != nil {
			errChan <- err
			return
		}

		done <- true
	}()

	select {
	case <-done:
		return nil
	case err := <-errChan:
		return err
	}
}

func extractYouTubeAudioURL(ytURL string) (string, error) {
	// Create a new video info fetcher
	vidInfo, err := ytdl.GetVideoInfo(ytURL)
	if err != nil {
		return "", err
	}

	// Choose the audio format with the highest quality
	var audioFormat *ytdl.Format
	for _, format := range vidInfo.Formats {
		if format.AudioEncoding == nil {
			continue
		}

		if audioFormat == nil || format.AudioQuality > audioFormat.AudioQuality {
			audioFormat = format
		}
	}

	if audioFormat == nil {
		return "", fmt.Errorf("no audio format found")
	}

	return audioFormat.URL, nil
}
