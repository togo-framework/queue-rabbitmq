package queuerabbitmq

import (
	"testing"

	"github.com/togo-framework/togo"
)

func TestNewRabbitFailsFastWithoutBroker(t *testing.T) {
	// with no broker reachable, newRabbit returns an error (not a panic)
	k := togo.New()
	t.Setenv("RABBITMQ_URL", "amqp://guest:guest@127.0.0.1:1/") // unreachable port
	if _, err := newRabbit(k); err == nil {
		t.Fatal("expected connect error without a broker")
	}
}
