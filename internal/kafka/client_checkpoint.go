package kafka

import (
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// XResumeTokenHeaderName corresponds to the header containing the MongoDB change stream
// resume token of the event (extended JSON, usable as MONGODB_OPTION_RESUME_AFTER)
const XResumeTokenHeaderName = "x-resume-token"

type trackFunc func(resumeToken []byte)

type clientCheckpoint struct {
	client Client
	track  trackFunc
}

// NewClientCheckpoint returns a kafka client that adds the resume token header on messages
// and tracks them before production, so their delivery reports can be checkpointed
func NewClientCheckpoint(cli Client, track trackFunc) *clientCheckpoint {
	return &clientCheckpoint{
		client: cli,
		track:  track,
	}
}

// Produce tracks the message resume token and then produces it
func (c *clientCheckpoint) Produce(messages chan *Message) {
	var next = make(chan *Message, len(messages))
	go func() {
		defer close(next)
		for message := range messages {
			if len(message.ResumeToken) > 0 {
				message.Headers = append(message.Headers, Header{
					Key:   XResumeTokenHeaderName,
					Value: message.ResumeToken,
				})
				c.track(message.ResumeToken)
			}
			next <- message
		}
	}()

	c.client.Produce(next)
}

// Events returns the kafka producer events
func (c *clientCheckpoint) Events() chan kafka.Event {
	return c.client.Events()
}

func (c *clientCheckpoint) Close() {
	c.client.Close()
}

// ResumeTokenFromHeaders returns the resume token header value of a delivered message
func ResumeTokenFromHeaders(headers []kafka.Header) []byte {
	for _, header := range headers {
		if header.Key == XResumeTokenHeaderName {
			return header.Value
		}
	}
	return nil
}
