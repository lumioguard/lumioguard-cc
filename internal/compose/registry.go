package compose

import (
	"github.com/lumioguard/lumioguard-cc/internal/adapter"
	"github.com/lumioguard/lumioguard-cc/internal/adapter/cfamily"
	"github.com/lumioguard/lumioguard-cc/internal/adapter/golang"
	"github.com/lumioguard/lumioguard-cc/internal/adapter/java"
	"github.com/lumioguard/lumioguard-cc/internal/adapter/python"
	"github.com/lumioguard/lumioguard-cc/internal/adapter/typescript"
)

// NewRegistry lists the installed language adapters in priority order.
func NewRegistry() *adapter.Registry {
	return adapter.NewRegistry(typescript.New(), python.New(), java.New(), golang.New(), cfamily.New())
}
