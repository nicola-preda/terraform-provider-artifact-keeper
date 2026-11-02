package provider

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nicola-preda/terraform-provider-artifact-keeper/internal/client"
)

// The data source decodes into repositoryResourceModel, so every model field
// needs a schema attribute or reads fail with a Value Conversion Error.
func TestRepositoryDataSourceSchemaCoversModel(t *testing.T) {
	var resp datasource.SchemaResponse
	NewRepositoryDataSource().Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	typ := reflect.TypeOf(repositoryResourceModel{})
	for i := 0; i < typ.NumField(); i++ {
		tag := typ.Field(i).Tag.Get("tfsdk")
		if _, ok := resp.Schema.Attributes[tag]; !ok {
			t.Errorf("data source schema lacks %q", tag)
		}
	}
}

// A zero-value list in the model has no element type, which the framework
// refuses on read ("MISSING TYPE"), so every list the mapper sets must be typed.
func TestRepositoryToModelListsAreTyped(t *testing.T) {
	m, diags := repositoryToModel(context.Background(), &client.Repository{})
	if diags.HasError() {
		t.Fatal(diags)
	}
	v := reflect.ValueOf(m)
	for i := 0; i < v.NumField(); i++ {
		if l, ok := v.Field(i).Interface().(types.List); ok && l.ElementType(context.Background()) == nil {
			t.Errorf("%s is an untyped list", v.Type().Field(i).Name)
		}
	}
}
