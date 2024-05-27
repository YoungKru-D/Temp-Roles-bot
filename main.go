package main

import (
	"fmt"
	"os"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

var (
	Token               string
	messageIDToRole     = make(map[string][]string) // map messageID to roleIDs
	messageIDToContent  = make(map[string]string)   // map messageID to its content
	messageIDToDuration = make(map[string]int)      // map messageID to duration
)

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file:", err)
		return
	}

	Token = os.Getenv("DISCORD_BOT_TOKEN")
	if Token == "" {
		fmt.Println("DISCORD_BOT_TOKEN is not set in .env file")
		return
	}

	dg, err := discordgo.New("Bot " + Token)
	if err != nil {
		fmt.Println("Error creating Discord session:", err)
		return
	}

	dg.AddHandler(ready)
	dg.AddHandler(interactionCreate)
	dg.AddHandler(messageReactionAdd)

	err = dg.Open()
	if err != nil {
		fmt.Println("Error opening Discord session:", err)
		return
	}
	defer dg.Close()

	command := &discordgo.ApplicationCommand{
		Name:        "temproles",
		Description: "Assign temporary roles via reactions",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "role1",
				Description: "First role",
				Type:        discordgo.ApplicationCommandOptionRole,
				Required:    true,
			},
			{
				Name:        "message_id",
				Description: "ID of the message to add reactions to",
				Type:        discordgo.ApplicationCommandOptionString,
				Required:    true,
			},
			{
				Name:        "duration",
				Description: "Duration for the temporary role in hours",
				Type:        discordgo.ApplicationCommandOptionInteger,
				Required:    true,
			},
			{
				Name:        "role2",
				Description: "Second role",
				Type:        discordgo.ApplicationCommandOptionRole,
				Required:    false,
			},
			{
				Name:        "role3",
				Description: "Third role",
				Type:        discordgo.ApplicationCommandOptionRole,
				Required:    false,
			},
			{
				Name:        "role4",
				Description: "Fourth role",
				Type:        discordgo.ApplicationCommandOptionRole,
				Required:    false,
			},
			{
				Name:        "role5",
				Description: "Fifth role",
				Type:        discordgo.ApplicationCommandOptionRole,
				Required:    false,
			},
		},
	}

	_, err = dg.ApplicationCommandCreate(dg.State.User.ID, "", command)
	if err != nil {
		fmt.Println("Error creating command:", err)
		return
	}

	fmt.Println("Bot is running. Press CTRL+C to exit.")
	select {}
}

func ready(s *discordgo.Session, event *discordgo.Ready) {
	s.UpdateGameStatus(0, "/temproles")
}

func interactionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type == discordgo.InteractionApplicationCommand {
		switch i.ApplicationCommandData().Name {
		case "temproles":
			handleTempRolesCommand(s, i)
		}
	}
}

func handleTempRolesCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	if len(options) < 3 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Usage: /temproles role1 message_id duration [role2] [role3] [role4] [role5]",
			},
		})
		return
	}

	guildID := i.GuildID
	var roleIDs []string
	var messageID string
	var duration int

	for _, option := range options {
		if option.Type == discordgo.ApplicationCommandOptionRole {
			role := option.RoleValue(s, guildID)
			if role == nil {
				fmt.Println("Error fetching role")
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Error fetching role",
					},
				})
				return
			}
			roleIDs = append(roleIDs, role.ID)
		} else if option.Type == discordgo.ApplicationCommandOptionString {
			messageID = option.StringValue()
		} else if option.Type == discordgo.ApplicationCommandOptionInteger {
			duration = int(option.IntValue())
		}
	}

	if messageID == "" || duration <= 0 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Message ID and duration are required",
			},
		})
		return
	}

	// Retrieve the message content to store it along with the role IDs
	message, err := s.ChannelMessage(i.ChannelID, messageID)
	if err != nil {
		fmt.Println("Error fetching message:", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Error fetching message content",
			},
		})
		return
	}

	// Store the messageID, roleIDs, message content, and duration in the maps
	messageIDToRole[messageID] = roleIDs
	messageIDToContent[messageID] = message.Content
	messageIDToDuration[messageID] = duration

	handleMessage(s, i, messageID, roleIDs)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Reactions added. Users can now react to get temporary roles.",
		},
	})
}

func handleMessage(s *discordgo.Session, i *discordgo.InteractionCreate, messageID string, roleIDs []string) {
	emojis := []string{"1️⃣", "2️⃣", "3️⃣", "4️⃣", "5️⃣"}

	for k := range roleIDs {
		if k < len(emojis) {
			err := s.MessageReactionAdd(i.ChannelID, messageID, emojis[k])
			if err != nil {
				fmt.Println("Error adding reaction:", err)
			}
		}
	}
}

func messageReactionAdd(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	messageID := r.MessageID

	fmt.Println("Reaction added by user:", r.UserID, "Emoji:", r.Emoji.Name, "MessageID:", messageID)

	// Retrieve roleIDs and message content from the maps using the messageID
	roleIDs, ok := messageIDToRole[messageID]
	if !ok {
		fmt.Println("Message ID not found in map:", messageID)
		return
	}

	duration, ok := messageIDToDuration[messageID]
	if !ok {
		fmt.Println("Duration not found for message ID:", messageID)
		duration = 1 // Default to 1 hour if duration is not found
	}

	fmt.Println("Message content for message ID:", messageID)

	var roleID string
	switch r.Emoji.Name {
	case "1️⃣":
		if len(roleIDs) >= 1 {
			roleID = roleIDs[0]
		}
	case "2️⃣":
		if len(roleIDs) >= 2 {
			roleID = roleIDs[1]
		}
	case "3️⃣":
		if len(roleIDs) >= 3 {
			roleID = roleIDs[2]
		}
	case "4️⃣":
		if len(roleIDs) >= 4 {
			roleID = roleIDs[3]
		}
	case "5️⃣":
		if len(roleIDs) >= 5 {
			roleID = roleIDs[4]
		}
	default:
		fmt.Println("Unknown emoji:", r.Emoji.Name)
		return
	}

	if roleID == "" {
		fmt.Println("Role ID not found for emoji:", r.Emoji.Name)
		return
	}

	channel, err := s.State.Channel(r.ChannelID)
	if err != nil {
		fmt.Println("Error fetching channel:", err)
		return
	}
	guildID := channel.GuildID

	fmt.Println("Adding role:", roleID, "to user:", r.UserID, "in guild:", guildID)

	err = s.GuildMemberRoleAdd(guildID, r.UserID, roleID)
	if err != nil {
		fmt.Println("Error adding role:", err)
		return
	}

	fmt.Println("Role added. Will remove after", duration, "hours")

	go func() {
		time.Sleep(time.Duration(duration) * time.Hour)
		err = s.GuildMemberRoleRemove(guildID, r.UserID, roleID)
		if err != nil {
			fmt.Println("Error removing role:", err)
		}
		fmt.Println("Role removed:", roleID, "from user:", r.UserID)
	}()
}
