package app

import (
	"errors"
	"fmt"

	"github.com/lumiostack/lumioguard-cc/internal/config"
	"github.com/lumiostack/lumioguard-cc/internal/product"
)

// ErrConfigExists is returned when `init` finds an existing configuration.
var ErrConfigExists = fmt.Errorf("%s already exists; it was not overwritten", product.ConfigFileName)

// InitService creates the default configuration without overwriting anything
// and without installing hooks.
type InitService struct {
	config ConfigLoader
}

// NewInitService creates an InitService.
func NewInitService(config ConfigLoader) *InitService {
	return &InitService{config: config}
}

// Initialize writes the default configuration file and returns its path.
func (s *InitService) Initialize(root string) (string, error) {
	path, err := s.config.WriteDefault(root)
	if errors.Is(err, config.ErrExists) {
		return path, ErrConfigExists
	}
	return path, err
}
