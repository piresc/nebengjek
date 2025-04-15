package gateway

import (
	"encoding/json"
	"fmt"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

// PublishBeaconEvent publishes a beacon event to NATS
func (g *UserGW) PublishBeaconEvent(event *models.BeaconEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	fmt.Printf("Publishing beacon event: %s\n", string(data))
	return g.nc.Publish("user.beacon", data)
}
