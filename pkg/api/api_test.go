package api_test

import (
	"context"
	"testing"

	"github.com/terminaldotshop/terminal-sdk-go"
	"github.com/terminaldotshop/terminal-sdk-go/option"
	"github.com/terminaldotshop/terminal/go/pkg/resource"
)

func TestGetProduct(t *testing.T) {
	client := terminal.NewClient(
		option.WithBaseURL(resource.Resource.Api.Url),
	)
	ctx := context.Background()
	client.Product.List(ctx)
}
