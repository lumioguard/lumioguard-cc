package app

import (
	"fmt"

	"github.com/lumiostack/lumioguard-cc/internal/guide"
	"github.com/lumiostack/lumioguard-cc/internal/product"
)

// GuideService serves the task guides for the installed version.
type GuideService struct {
	version string
}

// NewGuideService creates a GuideService.
func NewGuideService(version string) *GuideService {
	return &GuideService{version: version}
}

// Topics lists every guide.
func (s *GuideService) Topics() []guide.Topic {
	return guide.Topics()
}

// Topic returns one guide by name.
func (s *GuideService) Topic(name string) (guide.Topic, error) {
	topic, ok := guide.Lookup(name, s.version)
	if !ok {
		return guide.Topic{}, fmt.Errorf("unknown guide '%s'; run `%s guide` to list them", name, product.Name)
	}
	return topic, nil
}
