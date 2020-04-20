package main

import (
	"encoding/json"
	"errors"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"log"
	"os"
	"strings"
)

var (
	ErrNotPublicChannel = errors.New("not a public channel")
	ErrNotTimesChannel  = errors.New("not times channel")
	ErrBotMessage       = errors.New("sent by bot")
	ErrThreadMessage    = errors.New("thread")

	Token             = os.Getenv("TOKEN")
	VerificationToken = os.Getenv("VERIFICATION_TOKEN")
	TimelineChannel   = os.Getenv("TIMELINE_CHANNEL")
)

var api = slack.New(Token)

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	eventsAPIEvent, err := slackevents.ParseEvent(
		json.RawMessage(request.Body),
		slackevents.OptionVerifyToken(&slackevents.TokenComparator{VerificationToken: VerificationToken}),
	)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	if eventsAPIEvent.Type == slackevents.URLVerification {
		var r *slackevents.ChallengeResponse
		err := json.Unmarshal([]byte(request.Body), &r)
		if err != nil {
			return events.APIGatewayProxyResponse{}, err
		}

		return events.APIGatewayProxyResponse{
			StatusCode: 200,
			Headers:    map[string]string{"Content-Type": "text/plain"},
			Body:       r.Challenge,
		}, nil
	}

	if eventsAPIEvent.Type == slackevents.CallbackEvent {
		innerEvent := eventsAPIEvent.InnerEvent

		switch ev := innerEvent.Data.(type) {
		case *slackevents.MessageEvent:
			if ev.ChannelType != "channel" {
				return events.APIGatewayProxyResponse{}, ErrNotPublicChannel
			}

			if ev.BotID != "" {
				return events.APIGatewayProxyResponse{}, ErrBotMessage
			}

			if ev.ThreadTimeStamp != "" {
				return events.APIGatewayProxyResponse{}, ErrThreadMessage
			}

			channel, _ := api.GetConversationInfo(ev.Channel, false)

			if !strings.HasPrefix(channel.Name, "times-") {
				return events.APIGatewayProxyResponse{}, ErrNotTimesChannel
			}

			permalink, _ := api.GetPermalink(&slack.PermalinkParameters{
				Channel: ev.Channel,
				Ts:      ev.TimeStamp,
			})

			_, _, err = api.PostMessage(
				TimelineChannel,
				slack.MsgOptionText(permalink, false),
				slack.MsgOptionPostMessageParameters(slack.PostMessageParameters{
					UnfurlLinks: true,
					UnfurlMedia: true,
				}),
			)
			if err != nil {
				log.Println(err)
			}
		}
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
	}, nil
}

func main() {
	lambda.Start(handler)
}
