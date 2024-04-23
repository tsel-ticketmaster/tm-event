package event

import (
	"context"
	"encoding/json"
	"fmt"

	ck "github.com/confluentinc/confluent-kafka-go/kafka"
)

type OrderPaidEventHandler struct {
	EventUseCase EventUseCase
}

func (handler OrderPaidEventHandler) Handle(ctx context.Context, msg interface{}) error {
	kafkaMessage, ok := msg.(*ck.Message)
	if !ok {
		return fmt.Errorf("invalid message provider")
	}

	event := OrderPaidEvent{}
	json.Unmarshal(kafkaMessage.Value, &event)

	return handler.EventUseCase.OnOrderPaid(ctx, event)
}
