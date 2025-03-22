package loaders

import (
	"context"
	"reflect"

	"github.com/Gardego5/gocfg"
)

type withTag[T gocfg.Loader] struct {
	name   string
	loader T
}

func WithTag[T gocfg.Loader](name string, loader T) gocfg.Loader {
	return &withTag[T]{name: name, loader: loader}
}

func (w *withTag[T]) GocfgLoaderName() string { return w.name }
func (w *withTag[T]) Load(
	ctx context.Context,
	field reflect.StructField, value reflect.Value,
	resolvedTag string,
) error {
	return w.loader.Load(ctx, field, value, resolvedTag)
}
